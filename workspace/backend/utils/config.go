package utils

var AppConfig *Config

type Config struct {
	Server struct {
		Port  int    `yaml:"port"`
		Mode  string `yaml:"mode"`
		OrgId int    `yaml:"org_id"`
	} `yaml:"server"`
	Workers        []Worker       `yaml:"workers"`
	Database       PgDatabase     `yaml:"database"`
	GoogleSheetAPI GoogleSheetAPI `yaml:"google-sheet-api"`
	Connector      Connector      `yaml:"connector"`
	Kafka          struct {
		Producer KafkaProducerConfig `yaml:"producer"`
		Consumer KafkaConsumerConfig `yaml:"consumer"`
	} `yaml:"kafka"`
	Ecommerce struct {
		TiktokConfig TiktokConfig `yaml:"tiktok"`
		LazadaConfig LazadaConfig `yaml:"lazada"`
		ShopeeConfig ShopeeConfig `yaml:"shopee"`
	} `yaml:"ecommerce"`
	Logistic struct {
		Enable       bool   `yaml:"enable"`
		Host         string `yaml:"host"`
		ClientId     string `yaml:"client-id"`
		ClientSecret string `yaml:"client-secret"`
		FfmCode      string `yaml:"ffm-code"`
	} `yaml:"logistic"`
	Auth AuthConfig `yaml:"auth"`
}

// AuthConfig drives admin-app login (Google OIDC + internal email/password).
// Zero values are replaced by WithDefaults(). Never log this struct: it holds
// the Google client secret.
type AuthConfig struct {
	FrontendURL            string           `yaml:"frontend-url"` // redirect target after login and the only allowed Origin
	CookieName             string           `yaml:"cookie-name"`
	CookieSecure           bool             `yaml:"cookie-secure"` // true everywhere except local http
	SessionTTLHours        int              `yaml:"session-ttl-hours"`
	LoginAttemptTTLSeconds int              `yaml:"login-attempt-ttl-seconds"`
	MaxFailedLogins        int              `yaml:"max-failed-logins"`
	LockoutMinutes         int              `yaml:"lockout-minutes"`
	RememberMeDays         int              `yaml:"remember-me-days"`    // session length when "Ghi nhớ đăng nhập" is ticked
	MinPasswordLength      int              `yaml:"min-password-length"` // for new/changed passwords (not the seeded default admin)
	DefaultRole            string           `yaml:"default-role"`        // role code for auto-provisioned Google users and create-user without --role
	Google                 GoogleAuthConfig `yaml:"google"`
}

type GoogleAuthConfig struct {
	Enable         bool     `yaml:"enable"`
	ClientID       string   `yaml:"client-id"`
	ClientSecret   string   `yaml:"client-secret"` // server-side only
	RedirectURI    string   `yaml:"redirect-uri"`
	AllowedDomains []string `yaml:"allowed-domains"` // empty = any domain; checked against the hd claim AND the email suffix
	AutoProvision  bool     `yaml:"auto-provision"`  // create app_user on first Google login (after the domain check)
}

func (c AuthConfig) WithDefaults() AuthConfig {
	if c.FrontendURL == "" {
		c.FrontendURL = "http://localhost:8088"
	}
	if c.CookieName == "" {
		c.CookieName = "ecom_session"
	}
	if c.SessionTTLHours <= 0 {
		c.SessionTTLHours = 12
	}
	if c.LoginAttemptTTLSeconds <= 0 {
		c.LoginAttemptTTLSeconds = 600
	}
	if c.MaxFailedLogins <= 0 {
		c.MaxFailedLogins = 5
	}
	if c.LockoutMinutes <= 0 {
		c.LockoutMinutes = 15
	}
	if c.RememberMeDays <= 0 {
		c.RememberMeDays = 7
	}
	if c.MinPasswordLength <= 0 {
		c.MinPasswordLength = 10
	}
	if c.DefaultRole == "" {
		c.DefaultRole = "viewer"
	}
	return c
}

type PgDatabase struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	DbName      string `yaml:"dbname"`
	SslMode     string `yaml:"sslmode"`
	MaxOpenConn int    `yaml:"max-open-conn"`
	MaxIdleConn int    `yaml:"max-idle-conn"`
}

type GoogleSheetAPI struct {
	SpreadsheetID string `yaml:"spreadsheet-id"`
	SheetName     string `yaml:"sheetName"`
}

type Connector struct {
	Enable        bool   `yaml:"enable"` // also forward orders/DO updates to Connector (DB is always written)
	Authenticator string `yaml:"authenticator"`
	Host          string `yaml:"host"`
	Callback      string `yaml:"callback"`
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
}

type TiktokConfig struct {
	AppKey       string `yaml:"app_key"`
	AppSecret    string `yaml:"app_secret"`
	Url          string `yaml:"url"`
	ShopCipher   string `yaml:"shop_cipher"`
	RefreshToken string `yaml:"refresh_token"`
}

type ShopeeConfig struct {
	PartnerId    int    `yaml:"partner_id"`
	ShopId       int    `yaml:"shop_id"`
	Url          string `yaml:"url"`
	RefreshToken string `yaml:"refresh_token"`
	PartnerKey   string `yaml:"partner_key"`
}

type LazadaConfig struct {
	AppKey    string `yaml:"app_key"`
	Url       string `yaml:"url"`
	AppSecret string `yaml:"app_secret"`
}

type KafkaProducerConfig struct {
	BootstrapServer  string     `yaml:"bootstrap-servers"`
	SecurityProtocol string     `yaml:"security-protocol"`
	SaslMechanisms   string     `yaml:"sasl-mechanisms"`
	SaslUsername     string     `yaml:"sasl-username"`
	SaslPassword     string     `yaml:"sasl-password"`
	Acks             string     `yaml:"acks"`
	FooTopic         KafkaTopic `yaml:"topics-foo1"`
}

type KafkaConsumerConfig struct {
	BootstrapServer  string     `yaml:"bootstrap-servers"`
	SecurityProtocol string     `yaml:"security-protocol"`
	SaslMechanisms   string     `yaml:"sasl-mechanisms"`
	SaslUsername     string     `yaml:"sasl-username"`
	SaslPassword     string     `yaml:"sasl-password"`
	GroupId          string     `yaml:"group_id"`
	FooTopic         KafkaTopic `yaml:"topics-foo1"`
}

type KafkaTopic struct {
	Name      string `yaml:"name"`
	Partition int    `yaml:"partition"`
}

type Worker struct {
	Name     string `yaml:"name"`
	TimeRun  string `yaml:"time-run"`
	Function string `yaml:"function"`
	Enable   bool   `yaml:"enable"`
}
