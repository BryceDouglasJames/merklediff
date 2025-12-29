package reader

import (
	"strings"
	"testing"

	"github.com/BryceDouglasJames/merklediff/pkg/tree"
	"github.com/BryceDouglasJames/merklediff/pkg/types"
)

func TestCSVReader_Iterator(t *testing.T) {
	csvData := `id,name,email
1,Alice,alice@example.com
2,Bob,bob@example.com
3,Charlie,charlie@example.com`

	r, err := NewCSVReaderWithConfig(strings.NewReader(csvData), DefaultCSVConfig())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer r.Close()

	if len(r.Schema().Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(r.Schema().Columns))
	}

	count := 0
	for r.Next() {
		r.Row()
		count++
	}
	if err := r.Err(); err != nil {
		t.Fatalf("iteration: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows, got %d", count)
	}
}

func TestCSVReader_PrimaryKey(t *testing.T) {
	csvData := `user_id,name
U001,Alice
U002,Bob`

	config := CSVReaderConfig{KeyColumns: []int{0}, HasHeader: true}
	r, _ := NewCSVReaderWithConfig(strings.NewReader(csvData), config)
	defer r.Close()

	var keys []string
	for r.Next() {
		keys = append(keys, string(r.Row().Key))
	}

	if keys[0] != "U001" || keys[1] != "U002" {
		t.Fatalf("expected keys [U001, U002], got %v", keys)
	}
}

func TestCSVReader_CompositeKey(t *testing.T) {
	csvData := `region,store_id,product
US,101,Widget
EU,101,Gadget`

	config := CSVReaderConfig{KeyColumns: []int{0, 1}, HasHeader: true}
	r, _ := NewCSVReaderWithConfig(strings.NewReader(csvData), config)
	defer r.Close()

	var keys []string
	for r.Next() {
		keys = append(keys, string(r.Row().Key))
	}

	if keys[0] != "US:101" || keys[1] != "EU:101" {
		t.Fatalf("expected composite keys, got %v", keys)
	}
}

func TestCSVReader_FromPath(t *testing.T) {
	r, err := NewCSVReaderFromPath("testdata/sample.csv")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()

	count := 0
	for r.Next() {
		count++
	}
	if count != 20 {
		t.Fatalf("expected 20 rows, got %d", count)
	}
}

func TestCSVReader_IsSorted(t *testing.T) {
	config := CSVReaderConfig{HasHeader: true, IsSorted: true}
	r, _ := NewCSVReaderWithConfig(strings.NewReader("a,b\n1,2"), config)
	defer r.Close()

	if !r.IsSorted() {
		t.Fatal("expected IsSorted() = true")
	}
}

func TestCSVReader_CollectRows(t *testing.T) {
	csvData := `id,value
1,100
2,200
3,300`

	r, _ := NewCSVReaderWithConfig(strings.NewReader(csvData), DefaultCSVConfig())
	defer r.Close()

	rows, err := CollectRows(r)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestCSVReader_ImplementsRowReader(t *testing.T) {
	r, _ := NewCSVReader(strings.NewReader("a,b\n1,2"))
	defer r.Close()
	var _ types.RowReader = r // compile-time check
}

// ════════════════════════════════════════════════════════════════════════════
// Merkle Tree Integration
// ════════════════════════════════════════════════════════════════════════════

func TestCSVReader_MerkleTreeIntegration(t *testing.T) {
	csvA := `id,name,balance
1,Alice,100
2,Bob,200`

	csvB := `id,name,balance
1,Alice,100
2,Bob,250
3,Eve,500`

	config := CSVReaderConfig{KeyColumns: []int{0}, HasHeader: true}

	rA, _ := NewCSVReaderWithConfig(strings.NewReader(csvA), config)
	treeA, _ := tree.BuildTreeFromReader(rA)
	rA.Close()

	rB, _ := NewCSVReaderWithConfig(strings.NewReader(csvB), config)
	treeB, _ := tree.BuildTreeFromReader(rB)
	rB.Close()

	diff := tree.NewDiff(treeA, treeB)
	diff.Compare()

	if len(diff.GetRanges()) == 0 {
		t.Fatal("expected differences, got none")
	}
}

func TestCSVReader_MerkleTreeFromTestdata(t *testing.T) {
	config := CSVReaderConfig{KeyColumns: []int{0}, HasHeader: true}

	rA, _ := NewCSVReaderFromPathWithConfig("testdata/sample.csv", config)
	rowsA, _ := CollectRows(rA)

	rB, _ := NewCSVReaderFromPathWithConfig("testdata/sample_modified.csv", config)
	rowsB, _ := CollectRows(rB)

	treeA := tree.NewMerkleTreeFromRows(toTreeRows(rowsA))
	treeB := tree.NewMerkleTreeFromRows(toTreeRows(rowsB))

	diff := tree.NewDiff(treeA, treeB)
	diff.Compare()

	ranges := diff.GetRanges()
	t.Logf("Found %d differences between sample.csv and sample_modified.csv", len(ranges))

	if len(ranges) == 0 {
		t.Fatal("expected differences")
	}
}

func toTreeRows(rows []Row) []tree.Row {
	result := make([]tree.Row, len(rows))
	for i, r := range rows {
		result[i] = tree.Row{Key: r.Key, Values: r.Values}
	}
	return result
}
