package datagrid

import "fmt"

type deal struct {
	Name, Owner, Stage string
	Value              float64
	ID                 int
}

func deals(n int) []deal {
	stages := []string{"Lead", "Qualified", "Proposal", "Negotiation", "Won"}
	owners := []string{"Ana", "Bo", "Chen", "Dara", "Eli"}
	out := make([]deal, n)
	for i := range out {
		out[i] = deal{Name: fmt.Sprintf("Deal %d with a fairly long company name", i+1), Owner: owners[i%5], Stage: stages[(i*7)%5], Value: float64((i*7919)%100000) / 3, ID: i + 1}
	}
	return out
}

func dealValue(d deal, col string) any {
	switch col {
	case "id":
		return d.ID
	case "name":
		return d.Name
	case "owner":
		return d.Owner
	case "stage":
		return d.Stage
	case "value":
		return d.Value
	}
	return nil
}
