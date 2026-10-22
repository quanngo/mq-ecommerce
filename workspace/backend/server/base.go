package server

import (
	"context"
	"fmt"
	"main/component/handlers/rest"
	"main/component/repositories"
	"main/component/services"
	"main/internal"
	"main/middleware"
	"main/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v2"
)

var (
	configPath = "./config"
)

type apiServer struct {
	apiMiddleware    *middleware.ApiMiddleware
	cfg              *utils.Config
	lazadaService    services.LazadaService
	connectorService services.ConnectorService
	tiktokService    services.TikTokService
	authService      services.AuthService
	usersService     services.UserAdminService
	ordersService    services.OrderService
	imagesService    services.ProductImageService
}

type workerInstance struct {
	importSvc   services.ImportService
	logisticSvc services.LogisticService
	platformSvc services.PlatformService
	lazadaSvc   services.LazadaService
	tiktokSvc   services.TikTokService
	authSvc     services.AuthService
}

func initConfig() (*utils.Config, error) {
	config := &utils.Config{}

	if err := loadDotEnv(dotEnvPath()); err != nil {
		return nil, err
	}

	dir, err := os.ReadDir(configPath)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range dir {
		fileExtension := filepath.Ext(fileInfo.Name())
		if fileExtension != ".yml" {
			continue
		}
		file, err := os.Open(fmt.Sprintf("%s/%s", configPath, fileInfo.Name()))

		if err != nil {
			return nil, err
		}
		defer file.Close()

		d := yaml.NewDecoder(file)

		if err := d.Decode(&config); err != nil {
			return nil, err
		}
	}

	// Secrets come from ${VAR} placeholders (env or config/.env), see config_env.go.
	if missing := expandConfigEnv(config); len(missing) > 0 {
		utils.ShowInfoLogs("[initConfig] env vars referenced in config but not set (using empty): %s", strings.Join(missing, ", "))
	}

	return config, nil
}

func Initialize() *apiServer {
	var err error

	// config
	utils.AppConfig, err = initConfig()
	if err != nil {
		panic(err)
	}

	// init database
	dbInstance := internal.NewPgDatabase()
	db := dbInstance.ConnectPgDatabase(utils.AppConfig.Database)

	// init Kafka
	// producerTopics := []utils.KafkaTopic{utils.AppConfig.Kafka.Producer.FooTopic}
	// kafkaInstance := internal.NewKafkaInstance(producerTopics)

	// repos
	tokenRepo := repositories.NewTokenRepo(db)
	connectorRepo := repositories.NewConnectorRepo(db)
	marketPlaceMappingRepo := repositories.NewMarketPlaceMappingRepo(db)
	logMarketPlaceOrderRepo := repositories.NewLogMarketPlaceOrderRepo(db)
	ecomOrderRepo := repositories.NewEcomOrderRepo(db)

	// middleware
	apiMiddleware := middleware.NewMiddleware(utils.AppConfig, tokenRepo)

	// services
	connectorService := services.NewConnectorService(utils.AppConfig, apiMiddleware, ecomOrderRepo)
	logisticSvc := services.NewLogisticService(utils.AppConfig, apiMiddleware, connectorService)
	lazadaService := services.NewLazadaService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)
	tiktokService := services.NewTikTokService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)
	authService := newAuthService(db, utils.AppConfig.Auth, newGoogleClient(utils.AppConfig.Auth.Google))
	usersService := services.NewUserAdminService(repositories.NewUserRepo(db), repositories.NewRoleRepo(db), repositories.NewSessionRepo(db), authService)
	ordersService := services.NewOrderService(repositories.NewOrderQueryRepo(db))
	imagesService := services.NewProductImageService(apiMiddleware, tokenRepo, repositories.NewProductImageRepo(db))
	authService.WarnIfDefaultAdminPassword(context.Background())

	return &apiServer{
		apiMiddleware:    apiMiddleware,
		cfg:              utils.AppConfig,
		connectorService: connectorService,
		lazadaService:    lazadaService,
		tiktokService:    tiktokService,
		authService:      authService,
		usersService:     usersService,
		ordersService:    ordersService,
		imagesService:    imagesService,
	}
}

func newAuthService(db *sqlx.DB, cfg utils.AuthConfig, google services.OIDCClient) services.AuthService {
	return services.NewAuthService(cfg,
		repositories.NewUserRepo(db), repositories.NewSessionRepo(db), repositories.NewLoginAttemptRepo(db),
		google)
}

// newGoogleClient returns an untyped nil when Google login is off or discovery
// fails, so password login still works and Providers() reports google=false.
func newGoogleClient(cfg utils.GoogleAuthConfig) services.OIDCClient {
	if !cfg.Enable {
		return nil
	}
	client, err := services.NewGoogleOIDC(context.Background(), cfg)
	if err != nil {
		utils.ShowErrorLogs(fmt.Errorf("[Initialize] Google login disabled: %w", err))
		return nil
	}
	return client
}

func (apiServer *apiServer) Start() {
	apiServer.setMode(apiServer.cfg.Server.Mode)

	app := gin.New()
	//app.SetTrustedProxies(nil)
	app.Use(requestLogger())
	// app.Use(cors.Default())
	// /app.Use(...) //middleware

	rest.APIRoutes(app, &rest.ApiServer{
		ApiMiddleware: apiServer.apiMiddleware,
		Cfg:           apiServer.cfg,
		TiktokSvc:     apiServer.tiktokService,
		LazadaSvc:     apiServer.lazadaService,
		AuthSvc:       apiServer.authService,
		UsersSvc:      apiServer.usersService,
		OrdersSvc:     apiServer.ordersService,
		ImagesSvc:     apiServer.imagesService,
	})

	app.Run(fmt.Sprintf(":%d", apiServer.cfg.Server.Port))
}

// requestLogger is gin.Logger without the query string: the Google callback
// carries the OAuth code and state there.
func requestLogger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(p gin.LogFormatterParams) string {
			path := p.Path
			if i := strings.IndexByte(path, '?'); i >= 0 {
				path = path[:i]
			}
			return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n%s",
				p.TimeStamp.Format("2006/01/02 - 15:04:05"), p.StatusCode, p.Latency, p.ClientIP, p.Method, path, p.ErrorMessage)
		},
	})
}

func (apiServer *apiServer) setMode(mode string) {
	switch mode {
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	case "release":
		gin.SetMode(gin.ReleaseMode)
	}
}

func InitializeJob() {
	var err error

	// config
	utils.AppConfig, err = initConfig()
	if err != nil {
		panic(err)
	}

	// init database
	dbInstance := internal.NewPgDatabase()
	db := dbInstance.ConnectPgDatabase(utils.AppConfig.Database)

	// repos
	tokenRepo := repositories.NewTokenRepo(db)
	connectorRepo := repositories.NewConnectorRepo(db)
	marketPlaceMappingRepo := repositories.NewMarketPlaceMappingRepo(db)
	logMarketPlaceOrderRepo := repositories.NewLogMarketPlaceOrderRepo(db)
	ecomOrderRepo := repositories.NewEcomOrderRepo(db)

	// middleware
	apiMiddleware := middleware.NewMiddleware(utils.AppConfig, tokenRepo)

	// services
	connectorService := services.NewConnectorService(utils.AppConfig, apiMiddleware, ecomOrderRepo)
	logisticSvc := services.NewLogisticService(utils.AppConfig, apiMiddleware, connectorService)

	lazadaService := services.NewLazadaService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)
	lazadaService.CheckExpiredTokenLazada()
	tiktokService := services.NewTikTokService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)
	tiktokService.CheckExpiredTokenTiktok()

	importSvc := services.NewImportService(utils.AppConfig, connectorService)
	importSvc.ImportExcel()
}

func InitializeWorkers() *workerInstance {
	var err error

	// config
	utils.AppConfig, err = initConfig()
	if err != nil {
		panic(err)
	}

	// init database
	dbInstance := internal.NewPgDatabase()
	db := dbInstance.ConnectPgDatabase(utils.AppConfig.Database)

	// repos
	tokenRepo := repositories.NewTokenRepo(db)
	connectorRepo := repositories.NewConnectorRepo(db)
	marketPlaceMappingRepo := repositories.NewMarketPlaceMappingRepo(db)
	logMarketPlaceOrderRepo := repositories.NewLogMarketPlaceOrderRepo(db)
	ecomOrderRepo := repositories.NewEcomOrderRepo(db)
	logOrderStatusRepo := repositories.NewLogOrderStatusRepo(db)

	// middleware
	apiMiddleware := middleware.NewMiddleware(utils.AppConfig, tokenRepo)
	connectorService := services.NewConnectorService(utils.AppConfig, apiMiddleware, ecomOrderRepo)
	importSvc := services.NewImportService(utils.AppConfig, connectorService)
	logisticSvc := services.NewLogisticService(utils.AppConfig, apiMiddleware, connectorService)

	lazadaService := services.NewLazadaService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)
	tiktokService := services.NewTikTokService(utils.AppConfig, apiMiddleware, connectorRepo, tokenRepo, marketPlaceMappingRepo, logMarketPlaceOrderRepo, connectorService, logisticSvc)

	platformService := services.NewPlatformService(utils.AppConfig,
		connectorRepo, tokenRepo, marketPlaceMappingRepo, logOrderStatusRepo, logMarketPlaceOrderRepo,
		connectorService, tiktokService, lazadaService)

	return &workerInstance{
		importSvc:   importSvc,
		platformSvc: platformService,
		lazadaSvc:   lazadaService,
		tiktokSvc:   tiktokService,
		authSvc:     newAuthService(db, utils.AppConfig.Auth, nil), // cleanup only, no Google needed
	}

}
