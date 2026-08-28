package platformadmin

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomersHost interface {
	Pool() *pgxpool.Pool
	MapCustomerNotFound(err error) error
}

type Customers struct {
	host CustomersHost
}

func NewCustomers(host CustomersHost) *Customers {
	return &Customers{host: host}
}
