// Package ethdb provides batch extensions
package ethdb

// BatchWithDeleteRange extends Batch with DeleteRange support
type BatchWithDeleteRange interface {
	Batch
	DeleteRange(start, end []byte) error
}

// DeleteRange attempts to delete a range if the batch supports it
func DeleteRange(b Batch, start, end []byte) error {
	if br, ok := b.(BatchWithDeleteRange); ok {
		return br.DeleteRange(start, end)
	}
	// Fallback: not supported
	return nil
