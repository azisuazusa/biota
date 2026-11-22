package views

import (
	"slices"

	"github.com/azisuazusa/biota/services/sales"
)

// ChartData is the JSON shape read by static/js/charts.js.
type ChartData struct {
	Labels []string `json:"labels"`
	Values []int64  `json:"values"`
}

func channelChart(s *sales.Summary) ChartData {
	d := ChartData{Labels: []string{}, Values: []int64{}}
	for _, c := range s.ByChannel {
		d.Labels = append(d.Labels, string(c.Channel))
		d.Values = append(d.Values, c.NetRevenue)
	}
	return d
}

func topChart(s *sales.Summary) ChartData {
	d := ChartData{Labels: []string{}, Values: []int64{}}
	for _, p := range s.Top10 {
		d.Labels = append(d.Labels, p.DisplayName)
		d.Values = append(d.Values, p.NetRevenue)
	}
	return d
}

func dailyChart(s *sales.Summary) ChartData {
	d := ChartData{Labels: []string{}, Values: []int64{}}
	for _, p := range s.Daily {
		d.Labels = append(d.Labels, FormatDateID(p.Date))
		d.Values = append(d.Values, p.NetRevenue)
	}
	return d
}

func hasChannel(f sales.Filter, c sales.Channel) bool {
	return slices.Contains(f.Channels, c)
}

func dateValue(f sales.Filter, from bool) string {
	t := f.To
	if from {
		t = f.From
	}
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
