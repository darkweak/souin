package types

// CoalescingLayerStorage is the layer to be able to not coalesce uncoalesceable request
type CoalescingLayerStorage struct{}

// InitializeCoalescingLayerStorage initialize the storage
func InitializeCoalescingLayerStorage() *CoalescingLayerStorage {
	return &CoalescingLayerStorage{}
}

// Exists method returns if the key should coalesce
func (provider *CoalescingLayerStorage) Exists(key string) bool {
	return true
}

// Set method will store the response in Ristretto provider
func (provider *CoalescingLayerStorage) Set(key string) {}

// Delete method will delete the response in Ristretto provider if exists corresponding to key param
func (provider *CoalescingLayerStorage) Delete(key string) {}
