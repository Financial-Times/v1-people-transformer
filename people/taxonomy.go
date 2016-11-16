package main

type taxonomy struct {
	Terms []term `xml:"term"`
}

//TODO revise fields for people - Also need labels to come through too
type term struct {
	CanonicalName string `xml:"name"`
	RawID         string `xml:"id"`
}
