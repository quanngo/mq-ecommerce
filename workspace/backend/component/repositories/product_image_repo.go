package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"main/component/models"
	"strings"

	"github.com/jmoiron/sqlx"
)

// ProductImageRepo reads and writes public.ecom_product_image (EC-10).
type ProductImageRepo struct {
	db *sqlx.DB
}

func NewProductImageRepo(db *sqlx.DB) *ProductImageRepo {
	return &ProductImageRepo{
		db: db,
	}
}

const productImageSelect = `
	SELECT i.id, i.market_place, i.shop_id, i.use_case, i.uri, i.url, i.width, i.height,
	       i.file_name, i.size_bytes, i.uploaded_by, u.email AS uploaded_by_email, i.created_at
	FROM public.ecom_product_image i
	LEFT JOIN public.app_user u ON u.id = i.uploaded_by
`

// Insert stores an uploaded image and returns it as List would.
func (r *ProductImageRepo) Insert(ctx context.Context, img models.ProductImage) (*models.ProductImage, error) {
	strSql := `
		INSERT INTO public.ecom_product_image
		    (market_place, shop_id, use_case, uri, url, width, height, file_name, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	var id int64
	if err := r.db.QueryRowContext(ctx, strSql, img.MarketPlace, img.ShopID, img.UseCase, img.URI, img.URL,
		img.Width, img.Height, img.FileName, img.SizeBytes, img.UploadedBy).Scan(&id); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

// Get returns (nil, nil) when the image does not exist.
func (r *ProductImageRepo) Get(ctx context.Context, id int64) (*models.ProductImage, error) {
	var img models.ProductImage
	err := r.db.GetContext(ctx, &img, productImageSelect+` WHERE i.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// List returns one page of images, newest first, and the total that match.
func (r *ProductImageRepo) List(ctx context.Context, f models.ProductImageFilter) ([]models.ProductImage, int, error) {
	conditions := []string{}
	args := []interface{}{}
	if f.MarketPlace != "" {
		args = append(args, f.MarketPlace)
		conditions = append(conditions, fmt.Sprintf("i.market_place = $%d", len(args)))
	}
	if f.ShopID != "" {
		args = append(args, f.ShopID)
		conditions = append(conditions, fmt.Sprintf("i.shop_id = $%d", len(args)))
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM public.ecom_product_image i`+where, args...); err != nil {
		return nil, 0, err
	}
	images := []models.ProductImage{}
	pageArgs := append(append([]interface{}{}, args...), f.Limit, f.Offset)
	strSql := productImageSelect + where +
		fmt.Sprintf(" ORDER BY i.created_at DESC, i.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	if err := r.db.SelectContext(ctx, &images, strSql, pageArgs...); err != nil {
		return nil, 0, err
	}
	return images, total, nil
}
