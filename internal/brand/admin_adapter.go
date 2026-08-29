package brand

func NewAdminAdapter(store *Store) AdminService {
	if store == nil {
		return nil
	}
	return store
}
