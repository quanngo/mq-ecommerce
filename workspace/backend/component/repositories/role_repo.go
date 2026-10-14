package repositories

import (
	"context"
	"main/component/models"

	"github.com/jmoiron/sqlx"
)

// RoleRepo reads public.role / role_permission.
type RoleRepo struct {
	db *sqlx.DB
}

func NewRoleRepo(db *sqlx.DB) *RoleRepo {
	return &RoleRepo{
		db: db,
	}
}

// List returns every role with its permission codes, system roles first.
func (r *RoleRepo) List(ctx context.Context) ([]models.Role, error) {
	strSql := `
		SELECT r.id, r.code, r.name, r.description, r.is_system,
		       COALESCE(ARRAY(SELECT rp.permission_code FROM public.role_permission rp
		                      WHERE rp.role_id = r.id ORDER BY rp.permission_code), '{}') AS permissions
		FROM public.role r
		ORDER BY r.is_system DESC, r.id
	`
	roles := []models.Role{}
	if err := r.db.SelectContext(ctx, &roles, strSql); err != nil {
		return nil, err
	}
	return roles, nil
}
