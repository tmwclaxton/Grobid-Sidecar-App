package parsing

import (
	"testing"
)

type CrossRefResponse struct {
	Title    string
	Year     string
	DOI      string
	Abstract string
	ISSN     string
}

func checkField(t *testing.T, fieldName, expected, actual string) {
	if expected == actual {
		t.Logf("%s is correct", fieldName)
	} else {
		t.Errorf("%s is incorrect", fieldName)
	}
}

func TestCrossRefDataDOI(t *testing.T) {
	expectedResponse := CrossRefResponse{
		Title:    "Tebuconazole alters morphological, behavioral and neurochemical parameters in larvae and adult zebrafish (Danio rerio)",
		Year:     "2017",
		DOI:      "10.1016/j.chemosphere.2017.04.029",
		Abstract: "",
		ISSN:     "0045-6535",
	}

	response, err := CrossRefDataDOI(expectedResponse.DOI)

	if err != nil {
		t.Error(err)
	}

	checkField(t, "Title", expectedResponse.Title, response.Title)
	checkField(t, "Year", expectedResponse.Year, response.Year)
	checkField(t, "DOI", expectedResponse.DOI, response.DOI)
	checkField(t, "Abstract", expectedResponse.Abstract, response.Abstract)
	checkField(t, "ISSN", expectedResponse.ISSN, response.ISSN)
}

func TestCrossRefDataTitle(t *testing.T) {
	expectedResponse := CrossRefResponse{
		Title:    "Tebuconazole alters morphological, behavioral and neurochemical parameters in larvae and adult zebrafish (Danio rerio)",
		Year:     "2017",
		DOI:      "10.1016/j.chemosphere.2017.04.029",
		Abstract: "",
		ISSN:     "0045-6535",
	}

	response, err := CrossRefDataTitle(expectedResponse.Title)

	if err != nil {
		t.Error(err)
	}

	checkField(t, "Title", expectedResponse.Title, response.Title)
	checkField(t, "Year", expectedResponse.Year, response.Year)
	checkField(t, "DOI", expectedResponse.DOI, response.DOI)
	checkField(t, "Abstract", expectedResponse.Abstract, response.Abstract)
	checkField(t, "ISSN", expectedResponse.ISSN, response.ISSN)
}

func TestCrossRefData2(t *testing.T) {
	expectedResponse := CrossRefResponse{
		Title:    "The toxic effects of deltamethrin on Danio rerio: the correlation among behavior response, physiological damage and AChE",
		Year:     "2016",
		DOI:      "10.1039/c6ra23990k",
		Abstract: "<p>In this work we comprehensively evaluated the effects of deltamethrin, a pyrethroid pesticide, on the behavior, physiology and acetylcholinesterase (AChE) activity of fish.</p>",
		ISSN:     "2046-2069",
	}

	response, err := CrossRefDataTitle(expectedResponse.Title)

	if err != nil {
		t.Error(err)
	}

	checkField(t, "Title", expectedResponse.Title, response.Title)
	checkField(t, "Year", expectedResponse.Year, response.Year)
	checkField(t, "DOI", expectedResponse.DOI, response.DOI)
	checkField(t, "Abstract", expectedResponse.Abstract, response.Abstract)
	checkField(t, "ISSN", expectedResponse.ISSN, response.ISSN)

	response, err = CrossRefDataDOI(expectedResponse.DOI)

	if err != nil {
		t.Error(err)
	}

	checkField(t, "Title", expectedResponse.Title, response.Title)
	checkField(t, "Year", expectedResponse.Year, response.Year)
	checkField(t, "DOI", expectedResponse.DOI, response.DOI)
	checkField(t, "Abstract", expectedResponse.Abstract, response.Abstract)
	checkField(t, "ISSN", expectedResponse.ISSN, response.ISSN)
}
