package main

type SDNList struct {
	XMLName string `xml:"sdnList"`

	Entries []SDNEntry `xml:"sdnEntry"`
}
type SDNEntry struct {
	UID       string `xml:"uid"`
	FirstName string `xml:"firstName"`
	LastName  string `xml:"lastName"`
	SDNType   string `xml:"sdnType,omitempty"`
}

type APIAnswer struct {
	Result bool   `json:"result"`
	Info   string `json:"info"`
	Code   int    `json:"code,omitempty"`
}
