package data_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

func TestTable_CreateTableResponse(t *testing.T) {
	table := createTable(10)
	response := table.CreateTableResponse()

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := json.NewEncoder(w).Encode(response)
	require.NoError(t, err)
	_ = w.Flush()

	gp := filepath.Join("testdata", t.Name()+".golden")
	if *update {
		t.Logf("updating golden file for %s", t.Name())
		err = os.WriteFile(gp, b.Bytes(), 0644)
		require.NoError(t, err, "failed to update golden file")
	}

	var golden []byte
	golden, err = os.ReadFile(gp)
	require.NoError(t, err)

	assert.Equal(t, string(golden), b.String())
}
