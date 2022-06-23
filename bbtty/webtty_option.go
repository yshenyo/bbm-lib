package bbtty

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func (wt *WebTTY) SetPermitWrite(b bool) {
	wt.PermitWrite = b
}

func (wt *WebTTY) SetFixedColumns(columns int) {
	wt.Columns = columns
}

func (wt *WebTTY) SetFixedRows(rows int) {
	wt.Rows = rows
}

func (wt *WebTTY) SetWindowTitle(windowTitle []byte) {
	wt.WindowTitle = windowTitle
}

func (wt *WebTTY) SetReconnect(timeInSeconds int) {
	wt.Reconnect = timeInSeconds
}

func (wt *WebTTY) SetMasterPreferences(preferences interface{}) error {
	prefs, err := json.Marshal(preferences)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal preferences as JSON")
	}
	wt.MasterPrefs = prefs
	return nil
}
