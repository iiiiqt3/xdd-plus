package Xml

import "encoding/xml"

type TransferOperation struct {
	XMLName xml.Name `xml:"msg"`
	Appmsg  struct {
		XMLName   xml.Name `xml:"appmsg"`
		Wcpayinfo struct {
			XMLName           xml.Name `xml:"wcpayinfo"`
			Feedesc           string   `xml:"feedesc,CDATA"`
			Transcationid     string   `xml:"transcationid,CDATA"`
			Transferid        string   `xml:"transferid,CDATA"`
			Invalidtime       string   `xml:"invalidtime,CDATA"`
			Begintransfertime int64    `xml:"begintransfertime,CDATA"`
		}
	}
}
