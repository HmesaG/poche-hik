package reports

func companyNameOrDefault(companyName string) string {
	if companyName == "" || companyName == "Empresa" {
		return "Empresa S.R.L."
	}
	return companyName
}
