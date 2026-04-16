package reports

import (
	"fmt"
	"io"
	"ponches/internal/attendance"

	"github.com/xuri/excelize/v2"
)

// GenerateDailyExcel creates an Excel report for daily attendance
func GenerateDailyExcel(w io.Writer, companyName string, results []attendance.DayResult) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Asistencia"
	f.SetSheetName("Sheet1", sheet)

	// Company name header
	f.SetCellValue(sheet, "A1", "Empresa: "+companyNameOrDefault(companyName))
	f.MergeCell(sheet, "A1", "H1")

	// Title
	f.SetCellValue(sheet, "A2", "Reporte de Asistencia Diaria")
	f.MergeCell(sheet, "A2", "H2")

	// Headers
	headers := []string{"Empleado", "Fecha", "Entrada", "Salida", "Horas Totales", "Horas Extra", "Tarde", "Ausente"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 3)
		f.SetCellValue(sheet, cell, h)
	}

	// Style for header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A3", "H3", style)

	// Data
	for i, r := range results {
		row := i + 4
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.EmployeeNo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.Date.Format("2006-01-02"))

		if r.CheckIn != nil {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), r.CheckIn.Format("15:04"))
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "---")
		}

		if r.CheckOut != nil {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CheckOut.Format("15:04"))
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "---")
		}

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%.2f", r.TotalHours))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("%.2f", r.Overtime))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), r.IsLate)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), r.IsAbsent)
	}

	// Auto-fit columns
	f.SetColWidth(sheet, "A", "H", 15)

	return f.Write(w)
}

// GeneratePayrollExcel creates an Excel report for payroll
func GeneratePayrollExcel(w io.Writer, companyName string, results []attendance.PayrollResult) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Nomina"
	f.SetSheetName("Sheet1", sheet)

	// Company name header
	f.SetCellValue(sheet, "A1", "Empresa: "+companyNameOrDefault(companyName))
	f.MergeCell(sheet, "A1", "M1")

	// Title
	f.SetCellValue(sheet, "A2", "Reporte de Pre-Nomina")
	f.MergeCell(sheet, "A2", "M2")

	// Headers
	headers := []string{
		"Empleado", "No. Empleado", "Sueldo Base",
		"H. Extra Simple", "H. Extra Doble", "H. Extra Triple", "Total H. Extra",
		"Pago Extra", "Deducciones", "Comisiones", "Total a Pagar",
		"Dias Trabajados", "Dias Ausentes", "Dias Tarde",
	}
	for i, h := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 3)
		f.SetCellValue(sheet, cell, h)
	}

	// Style for header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A3", "N3", style)

	// Data
	for i, r := range results {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.EmployeeName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.EmployeeNo)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.2f", r.BaseSalary))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%.2f", r.OvertimeSimple))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%.2f", r.OvertimeDouble))
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("%.2f", r.OvertimeTriple))
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("%.2f", r.OvertimeHours))
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("%.2f", r.OvertimePay))
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("%.2f", r.Deductions))
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("%.2f", r.Commissions))
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("%.2f", r.TotalToPay))
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), r.DaysWorked)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), r.DaysAbsent)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), r.DaysLate)
	}

	// Auto-fit columns
	for col := 'A'; col <= 'N'; col++ {
		f.SetColWidth(sheet, string(col), string(col), 14)
	}

	// Add totals row
	totalRow := len(results) + 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), "TOTALES")

	// Sum formulas
	f.SetCellFormula(sheet, fmt.Sprintf("C%d", totalRow), fmt.Sprintf("SUM(C2:C%d)", len(results)+1))
	f.SetCellFormula(sheet, fmt.Sprintf("G%d", totalRow), fmt.Sprintf("SUM(G2:G%d)", len(results)+1))
	f.SetCellFormula(sheet, fmt.Sprintf("H%d", totalRow), fmt.Sprintf("SUM(H2:H%d)", len(results)+1))
	f.SetCellFormula(sheet, fmt.Sprintf("K%d", totalRow), fmt.Sprintf("SUM(K2:K%d)", len(results)+1))
	f.SetCellFormula(sheet, fmt.Sprintf("L%d", totalRow), fmt.Sprintf("SUM(L2:L%d)", len(results)+1))
	f.SetCellFormula(sheet, fmt.Sprintf("M%d", totalRow), fmt.Sprintf("SUM(M2:M%d)", len(results)+1))

	// Style for totals row
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFC000"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("N%d", totalRow), totalStyle)

	return f.Write(w)
}

// GenerateLateExcel creates an Excel report for late arrivals
func GenerateLateExcel(w io.Writer, companyName string, results []attendance.DayResult) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Tardanzas"
	f.SetSheetName("Sheet1", sheet)

	// Company name header
	f.SetCellValue(sheet, "A1", "Empresa: "+companyNameOrDefault(companyName))
	f.MergeCell(sheet, "A1", "F1")

	// Title
	f.SetCellValue(sheet, "A2", "Reporte de Tardanzas y Faltas")
	f.MergeCell(sheet, "A2", "F2")

	// Headers
	headers := []string{"Fecha", "Empleado", "Entrada Programada", "Entrada Real", "Minutos Tarde", "Tipo"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 3)
		f.SetCellValue(sheet, cell, h)
	}

	// Style for header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FF6B6B"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A3", "F3", style)

	// Data
	for i, r := range results {
		if !r.IsLate && !r.IsAbsent {
			continue
		}
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), r.Date.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), r.EmployeeNo)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "08:00") // Default shift start

		if r.CheckIn != nil {
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), r.CheckIn.Format("15:04"))
			if r.IsLate {
				// Calculate late minutes (simplified)
				f.SetCellValue(sheet, fmt.Sprintf("E%d", row), r.LateMinutes)
			}
		}

		if r.IsAbsent {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "FALTA")
		} else if r.IsLate {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "TARDE")
		}
	}

	// Auto-fit columns
	f.SetColWidth(sheet, "A", "F", 18)

	return f.Write(w)
}
