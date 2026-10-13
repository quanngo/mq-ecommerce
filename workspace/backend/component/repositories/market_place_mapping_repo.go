package repositories

import (
	"context"
	"fmt"
	"main/component/models"
	"main/utils"
	"strings"

	"github.com/jmoiron/sqlx"
)

var (
	WAREHOUSE             = "warehouse"
	PRODUCT               = "product"
	STATUS                = "status"
	PAYMENT_CONFIGURATION = "payment configuration"
)

type MarketPlaceMappingRepo struct {
	db *sqlx.DB
}

func NewMarketPlaceMappingRepo(db *sqlx.DB) *MarketPlaceMappingRepo {
	return &MarketPlaceMappingRepo{
		db: db,
	}
}

func (r *MarketPlaceMappingRepo) GetProducts(
	ctx context.Context,
	marketPlace utils.Plaform,
	shopID string,
	ecommerceIDs []string,
) ([]models.MarketPlaceMapping, error) {

	conditions := make([]string, 0)
	params := make([]interface{}, 0)
	var whereParams string
	index := 1

	index = utils.AppendStringCondition("type", PRODUCT, index, &conditions, &params)
	index = utils.AppendStringCondition("market_place", string(marketPlace), index, &conditions, &params)
	index = utils.AppendStringCondition("name", shopID, index, &conditions, &params)

	if len(ecommerceIDs) != 0 {
		ecommerceIDs = utils.CleanDuplicated(ecommerceIDs)

		conditionsIn := []string{}
		for _, id := range ecommerceIDs {
			params = append(params, id)
			conditionsIn = append(conditionsIn, fmt.Sprintf("$%d", index))
			index++
		}
		conditions = append(conditions, fmt.Sprintf("ecommerce_id IN (%s)", strings.Join(conditionsIn, ",")))
	}

	if len(conditions) > 0 {
		whereParams = " WHERE " + (strings.Join(conditions, " AND "))
	}

	query := `
        SELECT id, tms_id, ecommerce_id, name, type, market_place, org_id, quantity
        FROM public.market_place_mapping
    ` + whereParams

	var products []models.MarketPlaceMapping
	if err := r.db.SelectContext(ctx, &products, query, params...); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *MarketPlaceMappingRepo) GetStatus(ctx context.Context, marketPlace string) ([]models.MarketPlaceMapping, error) {

	strSql := `SELECT * FROM public.market_place_mapping 
			 WHERE type = 'status' 
			 AND market_place = $1`

	var statusList []models.MarketPlaceMapping
	if err := r.db.SelectContext(ctx, &statusList, strSql, marketPlace); err != nil {
		return nil, err
	}
	return statusList, nil
}

func (r *MarketPlaceMappingRepo) GetWarehouseByShopID(ctx context.Context, marketPlace utils.Plaform, shopID string) (*models.MarketPlaceMapping, error) {
	strSql := `SELECT * FROM public.market_place_mapping 
			 WHERE type = $1 
			 AND market_place = $2 
			 AND ecommerce_id = $3 
			 LIMIT 1 `
	var warehouse []*models.MarketPlaceMapping
	err := r.db.SelectContext(ctx, &warehouse, strSql, WAREHOUSE, marketPlace, shopID)
	if err != nil {
		return nil, err
	}
	return warehouse[0], nil
}

func (r *MarketPlaceMappingRepo) GetProductByProductName(ctx context.Context, marketPlace, shopID, name string) *models.MarketPlaceMapping {
	strSql := `SELECT * FROM public.market_place_mapping 
			 WHERE type = $1 
			 AND market_place = $2 
			 AND name = $3 
			 AND ecommerce_id = $4 
			 LIMIT 1 `
	var product []*models.MarketPlaceMapping
	err := r.db.SelectContext(ctx, &product, strSql, PRODUCT, marketPlace, shopID, name)
	if err != nil {
		return nil
	}
	if len(product) == 0 {
		return nil
	}

	return product[0]
}

func (r *MarketPlaceMappingRepo) GetShopPaymentConfig(ctx context.Context, platform utils.Plaform, shopId string) (string, error) {
	strSql := `SELECT tms_id FROM public.market_place_mapping 
			 WHERE type = $1 
			 AND market_place = $2 
			 AND ecommerce_id = $3 
			 LIMIT 1 `
	var orderId []string
	err := r.db.SelectContext(ctx, &orderId, strSql, PAYMENT_CONFIGURATION, platform, shopId)
	if err != nil {
		return "", err
	}
	if len(orderId) == 0 {
		return "", nil
	}

	return orderId[0], err
}

func (r *MarketPlaceMappingRepo) UpdateShopPaymentConfigValue(ctx context.Context, id int, platform utils.Plaform, shopId string) error {
	strSql := ` UPDATE public.market_place_mapping 
			 SET tms_id = $2
			 WHERE type = $1 
			 AND market_place = $3 
			 AND ecommerce_id = $4 
			 `
	_, err := r.db.ExecContext(ctx, strSql, PAYMENT_CONFIGURATION, fmt.Sprint(id), platform, shopId)
	return err
}
