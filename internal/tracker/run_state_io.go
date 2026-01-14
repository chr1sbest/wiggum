package tracker

import (
	"encoding/json"
	"os"
)

func (w *Writer) LoadRunState() (*RunState, error) {
	b, err := os.ReadFile(w.RunStatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rs RunState
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, nil
	}
	return &rs, nil
}
