package reports

import (
	"fmt"
	"io"
	"ponches/internal/attendance"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

// AttendanceRow is a single row in the period attendance report
type AttendanceRow struct {
	EmployeeNo   string
	EmployeeName string
	Department   string
	Date         time.Time
	CheckIn      *time.Time
	CheckOut     *time.Time
	TotalHours   float64
	OvertimeHrs  float64
	LateMinutes  int
	Status       string // Presente, Tarde, Falta, Incompleto
}

// BuildAttendanceRows converts DayResults (mapped per employee) into flat rows for the report
func BuildAttendanceRows(employeeResults map[string][]attendance.DayResult, nameMap, deptMap map[string]string) []AttendanceRow {
	var rows []AttendanceRow
	for empNo, days := range employeeResults {
		for _, day := range days {
			status := "Presente"
			if day.IsAbsent {
				status = "Falta"
			} else if day.IsIncomplete {
				status = "Incompleto"
			} else if day.IsLate {
				status = "Tarde"
			}
			rows = append(rows, AttendanceRow{
				EmployeeNo:   empNo,
				EmployeeName: nameMap[empNo],
				Department:   deptMap[empNo],
				Date:         day.Date,
				CheckIn:      day.CheckIn,
				CheckOut:     day.CheckOut,
				TotalHours:   day.TotalHours,
				OvertimeHrs:  day.Overtime,
				LateMinutes:  day.LateMinutes,
				Status:       status,
			})
		}
	}
	return rows
}

// GeneratePeriodAttendanceExcel creates the detailed period attendance Excel report
func GeneratePeriodAttendanceExcel(w io.Writer, companyName string, from, to time.Time, rows []AttendanceRow) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Asistencia"
	f.SetSheetName("Sheet1", sheet)

	// ── Styles ──────────────────────────────────────────────────────────────
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "1F3864"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	metaStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "444444"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1F6F5F"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "bottom", Color: "FFFFFF", Style: 2},
		},
	})

	presenteStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "1A7F60", Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	tardeStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "C17A00", Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	faltaStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "C0392B", Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	incompletoStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "7F6000", Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})

	dataLeftStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})

	totalLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// ── Row 1: Title ────────────────────────────────────────────────────────
	f.SetCellValue(sheet, "A1", "REPORTE DE ASISTENCIA POR PERÍODO")
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellStyle(sheet, "A1", "J1", titleStyle)
	f.SetRowHeight(sheet, 1, 28)

	// ── Row 2: Company ──────────────────────────────────────────────────────
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Empresa: %s", companyNameOrDefault(companyName)))
	f.MergeCell(sheet, "A2", "J2")
	f.SetCellStyle(sheet, "A2", "J2", metaStyle)

	// ── Row 3: Period ───────────────────────────────────────────────────────
	f.SetCellValue(sheet, "A3", fmt.Sprintf("Período: %s  al  %s", from.Format("02/01/2006"), to.Format("02/01/2006")))
	f.MergeCell(sheet, "A3", "J3")
	f.SetCellStyle(sheet, "A3", "J3", metaStyle)

	// ── Row 4: Printed ──────────────────────────────────────────────────────
	f.SetCellValue(sheet, "A4", fmt.Sprintf("Generado: %s", time.Now().Format("02/01/2006 15:04")))
	f.MergeCell(sheet, "A4", "J4")
	f.SetCellStyle(sheet, "A4", "J4", metaStyle)
	f.SetRowHeight(sheet, 4, 18)

	// ── Row 5: Summary ──────────────────────────────────────────────────────
	present := 0
	late := 0
	absent := 0
	incomplete := 0
	totalHoursAll := 0.0
	for _, r := range rows {
		switch r.Status {
		case "Presente":
			present++
		case "Tarde":
			late++
		case "Falta":
			absent++
		case "Incompleto":
			incomplete++
		}
		totalHoursAll += r.TotalHours
	}

	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#EAF4F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "1F6F5F", Style: 1},
			{Type: "bottom", Color: "1F6F5F", Style: 1},
		},
	})
	f.SetCellValue(sheet, "A5", fmt.Sprintf("Presentes: %d", present))
	f.SetCellValue(sheet, "C5", fmt.Sprintf("Tardanzas: %d", late))
	f.SetCellValue(sheet, "E5", fmt.Sprintf("Faltas: %d", absent))
	f.SetCellValue(sheet, "G5", fmt.Sprintf("Incompletos: %d", incomplete))
	f.SetCellValue(sheet, "I5", fmt.Sprintf("Horas Totales: %.2f", totalHoursAll))
	f.MergeCell(sheet, "A5", "B5")
	f.MergeCell(sheet, "C5", "D5")
	f.MergeCell(sheet, "E5", "F5")
	f.MergeCell(sheet, "G5", "H5")
	f.MergeCell(sheet, "I5", "J5")
	f.SetCellStyle(sheet, "A5", "J5", summaryStyle)
	f.SetRowHeight(sheet, 5, 22)

	// ── Row 6: Blank spacer ─────────────────────────────────────────────────
	f.SetRowHeight(sheet, 6, 10)

	// ── Row 7: Column headers ───────────────────────────────────────────────
	headers := []string{
		"No. Empleado", "Nombre", "Departamento", "Fecha",
		"Entrada", "Salida", "Horas", "Horas Extra", "Min. Tarde", "Estado",
	}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for i, h := range headers {
		f.SetCellValue(sheet, cols[i]+"7", h)
	}
	f.SetCellStyle(sheet, "A7", "J7", headerStyle)
	f.SetRowHeight(sheet, 7, 22)

	// ── Data rows (start at 8) ───────────────────────────────────────────────
	for i, r := range rows {
		rowNum := i + 8

		inStr := "---"
		if r.CheckIn != nil {
			inStr = r.CheckIn.Format("15:04")
		}
		outStr := "---"
		if r.CheckOut != nil {
			outStr = r.CheckOut.Format("15:04")
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), r.EmployeeNo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), r.EmployeeName)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), r.Department)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), r.Date.Format("02/01/2006"))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), inStr)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), outStr)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), r.TotalHours)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), r.OvertimeHrs)
		if r.LateMinutes > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), r.LateMinutes)
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), "---")
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", rowNum), r.Status)

		// Left-align name/dept
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), dataStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowNum), fmt.Sprintf("C%d", rowNum), dataLeftStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("I%d", rowNum), dataStyle)

		// Status colour
		switch r.Status {
		case "Presente":
			f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), presenteStyle)
		case "Tarde":
			f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), tardeStyle)
		case "Falta":
			f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), faltaStyle)
		default:
			f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), incompletoStyle)
		}
	}

	// ── Totals row ───────────────────────────────────────────────────────────
	totalRow := len(rows) + 8
	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), "TOTALES")
	f.MergeCell(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("F%d", totalRow))
	f.SetCellFormula(sheet, fmt.Sprintf("G%d", totalRow), fmt.Sprintf("SUM(G8:G%d)", totalRow-1))
	f.SetCellFormula(sheet, fmt.Sprintf("H%d", totalRow), fmt.Sprintf("SUM(H8:H%d)", totalRow-1))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("J%d", totalRow), totalLabelStyle)

	// ── Column widths ────────────────────────────────────────────────────────
	widths := map[string]float64{
		"A": 14, "B": 26, "C": 20, "D": 13,
		"E": 10, "F": 10, "G": 9, "H": 11, "I": 11, "J": 13,
	}
	for col, w := range widths {
		f.SetColWidth(sheet, col, col, w)
	}

	return f.Write(w)
}

// GeneratePeriodAttendancePDF creates the detailed period attendance PDF report
func GeneratePeriodAttendancePDF(w io.Writer, companyName string, from, to time.Time, rows []AttendanceRow) error {
	pdf := fpdf.New("L", "mm", "A4", "") // Landscape for more columns
	pdf.AddPage()
	pdf.SetMargins(10, 10, 10)

	pageW := 277.0 // A4 landscape usable width

	// ── Header ───────────────────────────────────────────────────────────────
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(pageW, 10, "REPORTE DE ASISTENCIA POR PERIODO", "0", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(pageW, 6, "Empresa: "+companyNameOrDefault(companyName), "0", 1, "L", false, 0, "")
	pdf.CellFormat(pageW, 6, fmt.Sprintf("Periodo: %s  al  %s", from.Format("02/01/2006"), to.Format("02/01/2006")), "0", 1, "L", false, 0, "")
	pdf.CellFormat(pageW, 6, fmt.Sprintf("Generado: %s", time.Now().Format("02/01/2006 15:04")), "0", 1, "L", false, 0, "")
	pdf.Ln(3)

	// ── Summary ───────────────────────────────────────────────────────────────
	present := 0
	late := 0
	absent := 0
	totalHrs := 0.0
	for _, r := range rows {
		switch r.Status {
		case "Presente":
			present++
		case "Tarde":
			late++
		case "Falta":
			absent++
		}
		totalHrs += r.TotalHours
	}
	pdf.SetFillColor(234, 244, 240)
	pdf.SetFont("Arial", "B", 9)
	bw := pageW / 4
	pdf.CellFormat(bw, 8, fmt.Sprintf("Presentes: %d", present), "1", 0, "C", true, 0, "")
	pdf.CellFormat(bw, 8, fmt.Sprintf("Tardanzas: %d", late), "1", 0, "C", true, 0, "")
	pdf.CellFormat(bw, 8, fmt.Sprintf("Faltas: %d", absent), "1", 0, "C", true, 0, "")
	pdf.CellFormat(bw, 8, fmt.Sprintf("Horas Totales: %.2f", totalHrs), "1", 1, "C", true, 0, "")
	pdf.Ln(4)

	// ── Table header ─────────────────────────────────────────────────────────
	pdf.SetFillColor(31, 111, 95)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)

	colW := []float64{18, 50, 35, 22, 18, 18, 18, 18, 20, 22}
	colH := []string{"No.", "Nombre", "Depto.", "Fecha", "Entrada", "Salida", "Horas", "H.Extra", "Min.Tard", "Estado"}

	for i, h := range colH {
		pdf.CellFormat(colW[i], 7, h, "0", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// ── Data rows ─────────────────────────────────────────────────────────────
	pdf.SetTextColor(40, 40, 40)
	pdf.SetFont("Arial", "", 8)

	for _, r := range rows {
		if pdf.GetY() > 185 { // Near bottom of page
			pdf.AddPage()
			pdf.SetFillColor(31, 111, 95)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Arial", "B", 8)
			for i, h := range colH {
				pdf.CellFormat(colW[i], 7, h, "0", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetTextColor(40, 40, 40)
			pdf.SetFont("Arial", "", 8)
		}

		inStr := "---"
		if r.CheckIn != nil {
			inStr = r.CheckIn.Format("15:04")
		}
		outStr := "---"
		if r.CheckOut != nil {
			outStr = r.CheckOut.Format("15:04")
		}
		lateStr := "---"
		if r.LateMinutes > 0 {
			lateStr = fmt.Sprintf("%d", r.LateMinutes)
		}

		pdf.CellFormat(colW[0], 6, r.EmployeeNo, "B", 0, "C", false, 0, "")
		pdf.CellFormat(colW[1], 6, r.EmployeeName, "B", 0, "L", false, 0, "")
		pdf.CellFormat(colW[2], 6, r.Department, "B", 0, "L", false, 0, "")
		pdf.CellFormat(colW[3], 6, r.Date.Format("02/01/2006"), "B", 0, "C", false, 0, "")
		pdf.CellFormat(colW[4], 6, inStr, "B", 0, "C", false, 0, "")
		pdf.CellFormat(colW[5], 6, outStr, "B", 0, "C", false, 0, "")
		pdf.CellFormat(colW[6], 6, fmt.Sprintf("%.2f", r.TotalHours), "B", 0, "R", false, 0, "")
		pdf.CellFormat(colW[7], 6, fmt.Sprintf("%.2f", r.OvertimeHrs), "B", 0, "R", false, 0, "")
		pdf.CellFormat(colW[8], 6, lateStr, "B", 0, "C", false, 0, "")
		pdf.CellFormat(colW[9], 6, r.Status, "B", 1, "C", false, 0, "")
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 7)
	pdf.Cell(pageW, 5, fmt.Sprintf("Generado por Sistema de Ponches - %s", time.Now().Format("02/01/2006 15:04")))

	return pdf.Output(w)
}
