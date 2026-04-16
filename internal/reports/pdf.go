package reports

import (
	"fmt"
	"io"
	"ponches/internal/attendance"
	"ponches/internal/employees"
	"time"

	"github.com/go-pdf/fpdf"
)

// GenerateDailyPDF creates a PDF report for daily attendance
func GenerateDailyPDF(w io.Writer, companyName string, results []attendance.DayResult) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Company Header
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(190, 10, "Reporte de Asistencia Diaria")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 6, "Empresa: "+companyNameOrDefault(companyName))
	pdf.Ln(6)

	// Date
	pdf.Cell(190, 6, fmt.Sprintf("Fecha: %s", time.Now().Format("02/01/2006")))
	pdf.Ln(8)

	// Summary
	present := 0
	late := 0
	absent := 0
	totalHours := 0.0

	for _, r := range results {
		if r.IsAbsent {
			absent++
		} else {
			present++
			totalHours += r.TotalHours
			if r.IsLate {
				late++
			}
		}
	}

	// Summary boxes
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(45, 8, "Presentes", "1", 0, "C", false, 0, "")
	pdf.CellFormat(45, 8, "Tardanzas", "1", 0, "C", false, 0, "")
	pdf.CellFormat(45, 8, "Ausentes", "1", 0, "C", false, 0, "")
	pdf.CellFormat(55, 8, "Horas Totales", "1", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(45, 8, fmt.Sprintf("%d", present), "1", 0, "C", false, 0, "")
	pdf.CellFormat(45, 8, fmt.Sprintf("%d", late), "1", 0, "C", false, 0, "")
	pdf.CellFormat(45, 8, fmt.Sprintf("%d", absent), "1", 0, "C", false, 0, "")
	pdf.CellFormat(55, 8, fmt.Sprintf("%.2f", totalHours), "1", 1, "C", false, 0, "")

	pdf.Ln(5)

	// Table header
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(50, 7, "Empleado", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 7, "Entrada", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 7, "Salida", "1", 0, "C", false, 0, "")
	pdf.CellFormat(25, 7, "Horas", "1", 0, "C", false, 0, "")
	pdf.CellFormat(25, 7, "Extra", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 7, "Estado", "1", 1, "C", false, 0, "")

	// Table body
	pdf.SetFont("Arial", "", 9)
	for _, r := range results {
		// Employee
		pdf.CellFormat(50, 6, r.EmployeeNo, "1", 0, "L", false, 0, "")

		// Check-in
		inStr := "---"
		if r.CheckIn != nil {
			inStr = r.CheckIn.Format("15:04")
		}
		pdf.CellFormat(30, 6, inStr, "1", 0, "C", false, 0, "")

		// Check-out
		outStr := "---"
		if r.CheckOut != nil {
			outStr = r.CheckOut.Format("15:04")
		}
		pdf.CellFormat(30, 6, outStr, "1", 0, "C", false, 0, "")

		// Hours
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", r.TotalHours), "1", 0, "R", false, 0, "")

		// Overtime
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", r.Overtime), "1", 0, "R", false, 0, "")

		// Status
		status := "Presente"
		if r.IsAbsent {
			status = "Ausente"
		} else if r.IsLate {
			status = "Tarde"
		}
		pdf.CellFormat(30, 6, status, "1", 1, "C", false, 0, "")
	}

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(190, 5, "Generado por Sistema de Ponches - "+time.Now().Format("02/01/2006 15:04"))

	return pdf.Output(w)
}

// GeneratePayrollPDF creates a PDF payroll report
func GeneratePayrollPDF(w io.Writer, companyName string, results []attendance.PayrollResult) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Company Header
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(190, 10, "Reporte de Pre-Nomina")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 6, "Empresa: "+companyNameOrDefault(companyName))
	pdf.Ln(6)

	// Period
	if len(results) > 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(190, 6, fmt.Sprintf("Periodo: %s al %s",
			results[0].PeriodFrom.Format("02/01/2006"),
			results[0].PeriodTo.Format("02/01/2006")))
		pdf.Ln(8)
	}

	// Summary
	totalBase := 0.0
	totalOvertime := 0.0
	totalToPay := 0.0

	for _, r := range results {
		totalBase += r.BaseSalary
		totalOvertime += r.OvertimePay
		totalToPay += r.TotalToPay
	}

	// Summary boxes
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(60, 8, "Sueldo Base", "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, "Horas Extra", "1", 0, "C", false, 0, "")
	pdf.CellFormat(70, 8, "Total a Pagar", "1", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(60, 8, fmt.Sprintf("$%.2f", totalBase), "1", 0, "C", false, 0, "")
	pdf.CellFormat(60, 8, fmt.Sprintf("$%.2f", totalOvertime), "1", 0, "C", false, 0, "")
	pdf.CellFormat(70, 8, fmt.Sprintf("$%.2f", totalToPay), "1", 1, "C", false, 0, "")

	pdf.Ln(5)

	// Table header
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 7, "Empleado", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Base", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, "H. Extra", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, "Pago Ext", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, "Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(25, 7, "Dias", "1", 1, "C", false, 0, "")

	// Table body
	pdf.SetFont("Arial", "", 9)
	for _, r := range results {
		// Employee name
		name := r.EmployeeName
		if name == "" {
			name = r.EmployeeNo
		}
		pdf.CellFormat(60, 6, name, "1", 0, "L", false, 0, "")

		// Base salary
		pdf.CellFormat(30, 6, fmt.Sprintf("$%.2f", r.BaseSalary), "1", 0, "R", false, 0, "")

		// Overtime hours
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", r.OvertimeHours), "1", 0, "R", false, 0, "")

		// Overtime pay
		pdf.CellFormat(25, 6, fmt.Sprintf("$%.2f", r.OvertimePay), "1", 0, "R", false, 0, "")

		// Total
		pdf.CellFormat(25, 6, fmt.Sprintf("$%.2f", r.TotalToPay), "1", 0, "R", false, 0, "")

		// Days worked
		pdf.CellFormat(25, 6, fmt.Sprintf("%d/%d", r.DaysWorked, r.DaysWorked+r.DaysAbsent), "1", 1, "C", false, 0, "")
	}

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(190, 5, "Generado por Sistema de Ponches - "+time.Now().Format("02/01/2006 15:04"))

	return pdf.Output(w)
}

// GenerateLateReportPDF creates a PDF report for late arrivals
func GenerateLateReportPDF(w io.Writer, companyName string, results []attendance.DayResult, from, to time.Time) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Company Header
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(190, 10, "Reporte de Tardanzas y Faltas")
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 6, "Empresa: "+companyNameOrDefault(companyName))
	pdf.Ln(6)

	// Period
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 6, fmt.Sprintf("Periodo: %s al %s", from.Format("02/01/2006"), to.Format("02/01/2006")))
	pdf.Ln(8)

	// Filter only late and absent
	var filtered []attendance.DayResult
	for _, r := range results {
		if r.IsLate || r.IsAbsent {
			filtered = append(filtered, r)
		}
	}

	// Summary
	lateCount := 0
	absentCount := 0
	for _, r := range filtered {
		if r.IsLate {
			lateCount++
		}
		if r.IsAbsent {
			absentCount++
		}
	}

	// Summary boxes
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(95, 8, fmt.Sprintf("Total Tardanzas: %d", lateCount), "1", 0, "C", false, 0, "")
	pdf.CellFormat(95, 8, fmt.Sprintf("Total Faltas: %d", absentCount), "1", 1, "C", false, 0, "")

	pdf.Ln(5)

	// Table header
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(30, 7, "Fecha", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 7, "Empleado", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, "Entrada", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 7, "Estado", "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 7, "Observaciones", "1", 1, "L", false, 0, "")

	// Table body
	pdf.SetFont("Arial", "", 9)
	for _, r := range filtered {
		// Date
		pdf.CellFormat(30, 6, r.Date.Format("02/01/2006"), "1", 0, "C", false, 0, "")

		// Employee
		pdf.CellFormat(50, 6, r.EmployeeNo, "1", 0, "L", false, 0, "")

		// Check-in
		inStr := "---"
		if r.CheckIn != nil {
			inStr = r.CheckIn.Format("15:04")
		}
		pdf.CellFormat(30, 6, inStr, "1", 0, "C", false, 0, "")

		// Status
		status := "Tarde"
		if r.IsAbsent {
			status = "Falta"
		}
		pdf.CellFormat(30, 6, status, "1", 0, "C", false, 0, "")

		// Notes
		notes := ""
		if r.IsLate && r.CheckIn != nil {
			notes = fmt.Sprintf("Retraso de minutos")
		}
		pdf.CellFormat(50, 6, notes, "1", 1, "L", false, 0, "")
	}

	// Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(190, 5, "Generado por Sistema de Ponches - "+time.Now().Format("02/01/2006 15:04"))

	return pdf.Output(w)
}

// GenerateTravelAllowancePDF creates a PDF voucher for a specific travel allowance request
func GenerateTravelAllowancePDF(w io.Writer, companyName string, ta *employees.TravelAllowance) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Company Header
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(190, 10, "VALE DE SOLICITUD DE VIÁTICOS", "0", 1, "C", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "I", 11)
	pdf.CellFormat(190, 6, companyNameOrDefault(companyName), "0", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Solicitud Info
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(190, 10, fmt.Sprintf("SOLICITUD #%s", ta.ID[:8]), "B", 1, "L", true, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, "Empleado:", "0", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(150, 8, ta.EmployeeName, "0", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, "Destino:", "0", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(150, 8, ta.Destination, "0", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, "Tarifa Base:", "0", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(150, 8, fmt.Sprintf("%s (%s)", ta.RateName, ta.RateType), "0", 1, "L", false, 0, "")

	pdf.Ln(5)

	// Financial Table
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(40, 8, "Fecha Salida", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Fecha Regreso", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Días", "1", 0, "C", true, 0, "")
	pdf.CellFormat(80, 8, "MONTO TOTAL RD$", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(40, 10, ta.DepartureDate.Format("02/01/2006"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 10, ta.ReturnDate.Format("02/01/2006"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, fmt.Sprintf("%d", ta.Days), "1", 0, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(80, 10, fmt.Sprintf("RD$ %.2f", ta.CalculatedAmount), "1", 1, "R", false, 0, "")

	pdf.Ln(8)

	// Reason / Notes
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(190, 8, "Motivo del Viaje:")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(190, 6, ta.Reason, "1", "L", false)

	if ta.Status != "Pending" {
		pdf.Ln(10)
		pdf.SetFont("Arial", "B", 11)
		label := "APROBACIÓN"
		if ta.Status == "Rejected" {
			label = "RECHAZO"
		}
		pdf.Cell(190, 8, "DETALLES DE " + label)
		pdf.Ln(8)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(40, 8, "Estado:")
		pdf.SetFont("Arial", "", 11)
		pdf.Cell(150, 8, translateStatus(ta.Status))
		pdf.Ln(8)
		
		if ta.Status == "Approved" {
			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(40, 8, "Aprobado por:")
			pdf.SetFont("Arial", "", 11)
			pdf.Cell(150, 8, ta.ApproverName)
			pdf.Ln(8)
		}

		if ta.ApprovalNotes != "" {
			pdf.SetFont("Arial", "B", 10)
			pdf.Cell(190, 8, "Notas:")
			pdf.Ln(8)
			pdf.SetFont("Arial", "", 10)
			pdf.MultiCell(190, 6, ta.ApprovalNotes, "1", "L", false)
		}
	}

	// Signatures
	pdf.SetY(-60)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(90, 5, "__________________________", "0", 0, "C", false, 0, "")
	pdf.CellFormat(10, 5, "", "0", 0, "C", false, 0, "")
	pdf.CellFormat(90, 5, "__________________________", "0", 1, "C", false, 0, "")

	pdf.CellFormat(90, 5, "Firma del Empleado", "0", 0, "C", false, 0, "")
	pdf.CellFormat(10, 5, "", "0", 0, "C", false, 0, "")
	pdf.CellFormat(90, 5, "Autorizado por", "0", 1, "C", false, 0, "")

	// Final Footer
	pdf.SetY(-20)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(190, 5, fmt.Sprintf("Fecha Impresión: %s - Generado por Ponches", time.Now().Format("02/01/2006 15:04")))

	return pdf.Output(w)
}

func translateStatus(s string) string {
	switch s {
	case "Pending":
		return "PENDIENTE"
	case "Approved":
		return "APROBADO"
	case "Rejected":
		return "RECHAZADO"
	default:
		return s
	}
}
