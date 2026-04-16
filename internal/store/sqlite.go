package store

import (
	"context"
	"database/sql"
	"fmt"
	"ponches/internal/employees"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			full_name TEXT NOT NULL,
			role TEXT DEFAULT 'viewer',
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS app_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS departments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			parent_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS positions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			department_id TEXT,
			level INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id)
		);`,
		`CREATE TABLE IF NOT EXISTS employees (
			id TEXT PRIMARY KEY,
			employee_no TEXT UNIQUE NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			id_number TEXT,
			gender TEXT,
			birth_date DATETIME,
			phone TEXT,
			email TEXT,
			department_id TEXT,
			position_id TEXT,
			hire_date DATETIME,
			status TEXT DEFAULT 'Active',
			base_salary REAL DEFAULT 0,
			face_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id),
			FOREIGN KEY (position_id) REFERENCES positions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS attendance_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			employee_no TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			type TEXT DEFAULT 'Access',
			raw_data TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS travel_allowance_rates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			value REAL NOT NULL,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS travel_allowances (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL,
			rate_id TEXT,
			group_id TEXT,
			group_name TEXT DEFAULT '',
			destination TEXT NOT NULL,
			departure_date DATETIME NOT NULL,
			return_date DATETIME NOT NULL,
			days INTEGER NOT NULL,
			reason TEXT,
			calculated_amount REAL NOT NULL,
			status TEXT DEFAULT 'Pending',
			approved_by TEXT,
			approval_notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (employee_id) REFERENCES employees(id),
			FOREIGN KEY (rate_id) REFERENCES travel_allowance_rates(id)
		);`,
		// Índices para mejorar performance
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_employees_employee_no ON employees(employee_no);`,
		`CREATE INDEX IF NOT EXISTS idx_employees_department ON employees(department_id);`,
		`CREATE INDEX IF NOT EXISTS idx_employees_status ON employees(status);`,
		`CREATE INDEX IF NOT EXISTS idx_attendance_employee_no ON attendance_events(employee_no);`,
		`CREATE INDEX IF NOT EXISTS idx_attendance_timestamp ON attendance_events(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_attendance_device ON attendance_events(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_travel_allowances_employee ON travel_allowances(employee_id);`,
		`CREATE INDEX IF NOT EXISTS idx_travel_allowances_status ON travel_allowances(status);`,
		`CREATE TABLE IF NOT EXISTS leaves (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL,
			type TEXT NOT NULL,
			start_date DATETIME NOT NULL,
			end_date DATETIME NOT NULL,
			days INTEGER NOT NULL DEFAULT 1,
			reason TEXT DEFAULT '',
			status TEXT DEFAULT 'Approved',
			authorized_by TEXT,
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (employee_id) REFERENCES employees(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_leaves_employee ON leaves(employee_id);`,
		`CREATE INDEX IF NOT EXISTS idx_leaves_dates ON leaves(start_date, end_date);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}
	if err := s.ensureColumn("departments", "description", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("departments", "manager_id", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("employees", "fleet_no", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("employees", "personal_no", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn("travel_allowances", "group_id", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("travel_allowances", "group_name", "TEXT DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ensureColumn(table, column, definition string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	if _, err := s.db.Exec(query); err != nil {
		errText := err.Error()
		if errText == "duplicate column name: "+column || errText == "SQL logic error: duplicate column name: "+column+" (1)" {
			return nil
		}
		return fmt.Errorf("ensure column %s.%s: %w", table, column, err)
	}
	return nil
}

func (s *SQLiteStore) CreateEmployee(ctx context.Context, e *employees.Employee) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO employees (id, employee_no, first_name, last_name, id_number, gender, birth_date,
			phone, email, department_id, position_id, hire_date, status, base_salary, face_id, fleet_no, personal_no)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.EmployeeNo, e.FirstName, e.LastName, e.IDNumber, e.Gender, e.BirthDate,
		e.Phone, e.Email, e.DepartmentID, e.PositionID, e.HireDate, e.Status, e.BaseSalary, e.FaceID,
		e.FleetNo, e.PersonalNo)
	return err
}

func (s *SQLiteStore) ListEmployees(ctx context.Context) ([]*employees.Employee, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, employee_no, first_name, last_name, id_number, gender, birth_date,
		       phone, email, department_id, position_id, hire_date, status, base_salary, face_id,
		       COALESCE(fleet_no,''), COALESCE(personal_no,'')
		FROM employees
		ORDER BY first_name, last_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.Employee
	for rows.Next() {
		e := &employees.Employee{}
		var birthDate, hireDate sql.NullTime
		var idNumber, gender, phone, email, faceID sql.NullString
		var baseSalary sql.NullFloat64

		err := rows.Scan(&e.ID, &e.EmployeeNo, &e.FirstName, &e.LastName, &idNumber, &gender,
			&birthDate, &phone, &email, &e.DepartmentID, &e.PositionID, &hireDate,
			&e.Status, &baseSalary, &faceID, &e.FleetNo, &e.PersonalNo)
		if err != nil {
			return nil, err
		}
		if birthDate.Valid {
			e.BirthDate = birthDate.Time
		}
		if hireDate.Valid {
			e.HireDate = hireDate.Time
		}
		if idNumber.Valid {
			e.IDNumber = idNumber.String
		}
		if gender.Valid {
			e.Gender = gender.String
		}
		if phone.Valid {
			e.Phone = phone.String
		}
		if email.Valid {
			e.Email = email.String
		}
		if faceID.Valid {
			e.FaceID = faceID.String
		}
		if baseSalary.Valid {
			e.BaseSalary = baseSalary.Float64
		}
		list = append(list, e)
	}
	return list, nil
}

// SaveEvent saves an attendance event to the database
func (s *SQLiteStore) SaveEvent(ctx context.Context, event *AttendanceEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO attendance_events (device_id, employee_no, timestamp, type) VALUES (?, ?, ?, ?)`,
		event.DeviceID, event.EmployeeNo, event.Timestamp, event.Type)
	return err
}

func (s *SQLiteStore) GetEmployee(ctx context.Context, id string) (*employees.Employee, error) {
	e := &employees.Employee{}
	var birthDate, hireDate sql.NullTime
	var idNumber, gender, phone, email, faceID sql.NullString
	var baseSalary sql.NullFloat64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, employee_no, first_name, last_name, id_number, gender, birth_date,
		        phone, email, department_id, position_id, hire_date, status, base_salary, face_id,
		        COALESCE(fleet_no,''), COALESCE(personal_no,'')
		 FROM employees WHERE id = ?`, id).
		Scan(&e.ID, &e.EmployeeNo, &e.FirstName, &e.LastName, &idNumber, &gender,
			&birthDate, &phone, &email, &e.DepartmentID, &e.PositionID, &hireDate,
			&e.Status, &baseSalary, &faceID, &e.FleetNo, &e.PersonalNo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if birthDate.Valid {
		e.BirthDate = birthDate.Time
	}
	if hireDate.Valid {
		e.HireDate = hireDate.Time
	}
	if idNumber.Valid {
		e.IDNumber = idNumber.String
	}
	if gender.Valid {
		e.Gender = gender.String
	}
	if phone.Valid {
		e.Phone = phone.String
	}
	if email.Valid {
		e.Email = email.String
	}
	if faceID.Valid {
		e.FaceID = faceID.String
	}
	if baseSalary.Valid {
		e.BaseSalary = baseSalary.Float64
	}
	return e, nil
}

func (s *SQLiteStore) GetEmployeeByNo(ctx context.Context, employeeNo string) (*employees.Employee, error) {
	e := &employees.Employee{}
	var birthDate, hireDate sql.NullTime
	var idNumber, gender, phone, email, faceID sql.NullString
	var baseSalary sql.NullFloat64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, employee_no, first_name, last_name, id_number, gender, birth_date,
		        phone, email, department_id, position_id, hire_date, status, base_salary, face_id,
		        COALESCE(fleet_no,''), COALESCE(personal_no,'')
		 FROM employees WHERE employee_no = ?`, employeeNo).
		Scan(&e.ID, &e.EmployeeNo, &e.FirstName, &e.LastName, &idNumber, &gender,
			&birthDate, &phone, &email, &e.DepartmentID, &e.PositionID, &hireDate,
			&e.Status, &baseSalary, &faceID, &e.FleetNo, &e.PersonalNo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if birthDate.Valid {
		e.BirthDate = birthDate.Time
	}
	if hireDate.Valid {
		e.HireDate = hireDate.Time
	}
	if idNumber.Valid {
		e.IDNumber = idNumber.String
	}
	if gender.Valid {
		e.Gender = gender.String
	}
	if phone.Valid {
		e.Phone = phone.String
	}
	if email.Valid {
		e.Email = email.String
	}
	if faceID.Valid {
		e.FaceID = faceID.String
	}
	if baseSalary.Valid {
		e.BaseSalary = baseSalary.Float64
	}
	return e, nil
}

func (s *SQLiteStore) UpdateEmployee(ctx context.Context, e *employees.Employee) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE employees SET
			employee_no = ?, first_name = ?, last_name = ?, id_number = ?, gender = ?, birth_date = ?,
			phone = ?, email = ?, department_id = ?, position_id = ?, hire_date = ?,
			status = ?, base_salary = ?, face_id = ?, fleet_no = ?, personal_no = ?,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		e.EmployeeNo, e.FirstName, e.LastName, e.IDNumber, e.Gender, e.BirthDate,
		e.Phone, e.Email, e.DepartmentID, e.PositionID, e.HireDate,
		e.Status, e.BaseSalary, e.FaceID, e.FleetNo, e.PersonalNo, e.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteEmployee(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM employees WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) CreateDepartment(ctx context.Context, d *employees.Department) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO departments (id, name, description, parent_id, manager_id) VALUES (?, ?, ?, ?, ?)`,
		d.ID, d.Name, d.Description, d.ParentID, d.ManagerID)
	return err
}

func (s *SQLiteStore) ListDepartments(ctx context.Context) ([]*employees.Department, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.name, COALESCE(d.description,''), COALESCE(d.parent_id,''),
		       COALESCE(d.manager_id,''), COALESCE(u.full_name,'')
		FROM departments d
		LEFT JOIN users u ON d.manager_id = u.id
		ORDER BY d.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.Department
	for rows.Next() {
		d := &employees.Department{}
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.ParentID, &d.ManagerID, &d.ManagerName); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (s *SQLiteStore) CreatePosition(ctx context.Context, p *employees.Position) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO positions (id, name, department_id, level) VALUES (?, ?, ?, ?)`,
		p.ID, p.Name, p.DepartmentID, p.Level)
	return err
}

func (s *SQLiteStore) ListPositions(ctx context.Context) ([]*employees.Position, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, department_id, level FROM positions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.Position
	for rows.Next() {
		p := &employees.Position{}
		if err := rows.Scan(&p.ID, &p.Name, &p.DepartmentID, &p.Level); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

func (s *SQLiteStore) UpsertDepartment(ctx context.Context, d *employees.Department) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO departments (id, name, description, parent_id, manager_id) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name = excluded.name, description = excluded.description,
		   parent_id = excluded.parent_id, manager_id = excluded.manager_id`,
		d.ID, d.Name, d.Description, d.ParentID, d.ManagerID)
	return err
}

func (s *SQLiteStore) GetDepartment(ctx context.Context, id string) (*employees.Department, error) {
	d := &employees.Department{}
	err := s.db.QueryRowContext(ctx,
		`SELECT d.id, d.name, COALESCE(d.description,''), COALESCE(d.parent_id,''),
		        COALESCE(d.manager_id,''), COALESCE(u.full_name,'')
		 FROM departments d LEFT JOIN users u ON d.manager_id = u.id
		 WHERE d.id = ?`, id).
		Scan(&d.ID, &d.Name, &d.Description, &d.ParentID, &d.ManagerID, &d.ManagerName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (s *SQLiteStore) UpdateDepartment(ctx context.Context, d *employees.Department) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE departments SET name = ?, description = ?, parent_id = ?, manager_id = ? WHERE id = ?`,
		d.Name, d.Description, d.ParentID, d.ManagerID, d.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteDepartment(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM departments WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) UpsertPosition(ctx context.Context, p *employees.Position) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO positions (id, name, department_id, level) VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name = excluded.name, department_id = excluded.department_id, level = excluded.level`,
		p.ID, p.Name, p.DepartmentID, p.Level)
	return err
}

func (s *SQLiteStore) GetPosition(ctx context.Context, id string) (*employees.Position, error) {
	p := &employees.Position{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, department_id, level FROM positions WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.DepartmentID, &p.Level)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *SQLiteStore) UpdatePosition(ctx context.Context, p *employees.Position) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE positions SET name = ?, department_id = ?, level = ? WHERE id = ?`,
		p.Name, p.DepartmentID, p.Level, p.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeletePosition(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM positions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) UpsertEmployee(ctx context.Context, e *employees.Employee) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO employees (id, employee_no, first_name, last_name, email, department_id, position_id, status, base_salary, face_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		 	employee_no = excluded.employee_no,
		 	first_name = excluded.first_name,
		 	last_name = excluded.last_name,
		 	email = excluded.email,
		 	department_id = excluded.department_id,
		 	position_id = excluded.position_id,
		 	status = excluded.status,
		 	base_salary = excluded.base_salary,
		 	face_id = excluded.face_id,
		 	updated_at = CURRENT_TIMESTAMP`,
		e.ID, e.EmployeeNo, e.FirstName, e.LastName, e.Email, e.DepartmentID, e.PositionID, e.Status, e.BaseSalary, e.FaceID)
	return err
}

func (s *SQLiteStore) GetEvents(ctx context.Context, filter EventFilter) ([]*AttendanceEvent, error) {
	query := `SELECT id, device_id, employee_no, timestamp, type FROM attendance_events WHERE 1=1`
	args := []interface{}{}

	if filter.EmployeeNo != "" {
		query += " AND employee_no = ?"
		args = append(args, filter.EmployeeNo)
	}
	if !filter.From.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.To)
	}
	query += " ORDER BY timestamp DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*AttendanceEvent
	for rows.Next() {
		ev := &AttendanceEvent{}
		if err := rows.Scan(&ev.ID, &ev.DeviceID, &ev.EmployeeNo, &ev.Timestamp, &ev.Type); err != nil {
			return nil, err
		}
		list = append(list, ev)
	}
	return list, nil
}

// ==================== TRAVEL ALLOWANCE RATES ====================

func (s *SQLiteStore) CreateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO travel_allowance_rates (id, name, type, value, active) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Type, r.Value, r.Active)
	return err
}

func (s *SQLiteStore) GetTravelRate(ctx context.Context, id string) (*employees.TravelAllowanceRate, error) {
	r := &employees.TravelAllowanceRate{}
	var active int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, type, value, active FROM travel_allowance_rates WHERE id = ?`, id).
		Scan(&r.ID, &r.Name, &r.Type, &r.Value, &active)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Active = active == 1
	return r, nil
}

func (s *SQLiteStore) ListTravelRates(ctx context.Context) ([]*employees.TravelAllowanceRate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, type, value, active FROM travel_allowance_rates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.TravelAllowanceRate
	for rows.Next() {
		r := &employees.TravelAllowanceRate{}
		var active int
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Value, &active); err != nil {
			return nil, err
		}
		r.Active = active == 1
		list = append(list, r)
	}
	return list, nil
}

func (s *SQLiteStore) UpdateTravelRate(ctx context.Context, r *employees.TravelAllowanceRate) error {
	var activeInt int
	if r.Active {
		activeInt = 1
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE travel_allowance_rates SET name = ?, type = ?, value = ?, active = ? WHERE id = ?`,
		r.Name, r.Type, r.Value, activeInt, r.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteTravelRate(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM travel_allowance_rates WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ==================== TRAVEL ALLOWANCES ====================

func (s *SQLiteStore) CreateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO travel_allowances (id, employee_id, rate_id, group_id, group_name, destination, departure_date, return_date,
			days, reason, calculated_amount, status, approved_by, approval_notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ta.ID, ta.EmployeeID, ta.RateID, ta.GroupID, ta.GroupName, ta.Destination, ta.DepartureDate, ta.ReturnDate,
		ta.Days, ta.Reason, ta.CalculatedAmount, ta.Status, ta.ApprovedBy, ta.ApprovalNotes)
	return err
}

func (s *SQLiteStore) GetTravelAllowance(ctx context.Context, id string) (*employees.TravelAllowance, error) {
	ta := &employees.TravelAllowance{}
	var approvedBy, approvalNotes, reason sql.NullString
	var rateID sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT ta.id, ta.employee_id,
			COALESCE(e.first_name || ' ' || e.last_name, '') as employee_name,
			ta.rate_id, COALESCE(r.name, '') as rate_name, COALESCE(r.type, '') as rate_type,
			COALESCE(ta.group_id, ''), COALESCE(ta.group_name, ''),
			ta.destination, ta.departure_date, ta.return_date, ta.days,
			ta.reason, ta.calculated_amount, ta.status, ta.approved_by,
			COALESCE(u.full_name, '') as approver_name,
			ta.approval_notes,
			ta.created_at, ta.updated_at
		FROM travel_allowances ta
		LEFT JOIN employees e ON ta.employee_id = e.id
		LEFT JOIN travel_allowance_rates r ON ta.rate_id = r.id
		LEFT JOIN users u ON ta.approved_by = u.id
		WHERE ta.id = ?`, id).
		Scan(&ta.ID, &ta.EmployeeID, &ta.EmployeeName, &rateID, &ta.RateName, &ta.RateType, &ta.GroupID, &ta.GroupName,
			&ta.Destination, &ta.DepartureDate, &ta.ReturnDate, &ta.Days,
			&reason, &ta.CalculatedAmount, &ta.Status, &approvedBy, &ta.ApproverName, &approvalNotes,
			&ta.CreatedAt, &ta.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rateID.Valid {
		ta.RateID = rateID.String
	}
	if reason.Valid {
		ta.Reason = reason.String
	}
	if approvedBy.Valid {
		ta.ApprovedBy = approvedBy.String
	}
	if approvalNotes.Valid {
		ta.ApprovalNotes = approvalNotes.String
	}
	return ta, nil
}

func (s *SQLiteStore) ListTravelAllowances(ctx context.Context) ([]*employees.TravelAllowance, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ta.id, ta.employee_id,
			COALESCE(e.first_name || ' ' || e.last_name, '') as employee_name,
			ta.rate_id, COALESCE(r.name, '') as rate_name, COALESCE(r.type, '') as rate_type,
			COALESCE(ta.group_id, ''), COALESCE(ta.group_name, ''),
			ta.destination, ta.departure_date, ta.return_date, ta.days,
			ta.reason, ta.calculated_amount, ta.status, ta.approved_by,
			COALESCE(u.full_name, '') as approver_name,
			ta.approval_notes,
			ta.created_at, ta.updated_at
		FROM travel_allowances ta
		LEFT JOIN employees e ON ta.employee_id = e.id
		LEFT JOIN travel_allowance_rates r ON ta.rate_id = r.id
		LEFT JOIN users u ON ta.approved_by = u.id
		ORDER BY ta.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.TravelAllowance
	for rows.Next() {
		ta := &employees.TravelAllowance{}
		var approvedBy, approvalNotes, reason sql.NullString
		var rateID sql.NullString

		err := rows.Scan(&ta.ID, &ta.EmployeeID, &ta.EmployeeName, &rateID, &ta.RateName, &ta.RateType, &ta.GroupID, &ta.GroupName,
			&ta.Destination, &ta.DepartureDate, &ta.ReturnDate, &ta.Days,
			&reason, &ta.CalculatedAmount, &ta.Status, &approvedBy, &ta.ApproverName, &approvalNotes,
			&ta.CreatedAt, &ta.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if rateID.Valid {
			ta.RateID = rateID.String
		}
		if reason.Valid {
			ta.Reason = reason.String
		}
		if approvedBy.Valid {
			ta.ApprovedBy = approvedBy.String
		}
		if approvalNotes.Valid {
			ta.ApprovalNotes = approvalNotes.String
		}
		list = append(list, ta)
	}
	return list, nil
}

func (s *SQLiteStore) UpdateTravelAllowance(ctx context.Context, ta *employees.TravelAllowance) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE travel_allowances SET
			employee_id = ?, rate_id = ?, group_id = ?, group_name = ?, destination = ?, departure_date = ?, return_date = ?,
			days = ?, reason = ?, calculated_amount = ?, status = ?,
			approved_by = ?, approval_notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		ta.EmployeeID, ta.RateID, ta.GroupID, ta.GroupName, ta.Destination, ta.DepartureDate, ta.ReturnDate,
		ta.Days, ta.Reason, ta.CalculatedAmount, ta.Status,
		ta.ApprovedBy, ta.ApprovalNotes, ta.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteTravelAllowance(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM travel_allowances WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ==================== LEAVES (PERMISOS Y AUSENCIAS) ====================

func (s *SQLiteStore) CreateLeave(ctx context.Context, l *employees.Leave) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO leaves (id, employee_id, type, start_date, end_date, days, reason, status, authorized_by, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.EmployeeID, l.Type, l.StartDate, l.EndDate, l.Days, l.Reason, l.Status, l.AuthorizedBy, l.Notes)
	return err
}

func (s *SQLiteStore) GetLeave(ctx context.Context, id string) (*employees.Leave, error) {
	l := &employees.Leave{}
	var authorizedBy, notes, reason sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT l.id, l.employee_id,
			COALESCE(e.first_name||' '||e.last_name,'') as employee_name,
			COALESCE(e.employee_no,'') as employee_no,
			COALESCE(d.name,'') as department,
			l.type, l.start_date, l.end_date, l.days, l.reason, l.status,
			l.authorized_by, COALESCE(u.full_name,'') as authorizer_name,
			l.notes, l.created_at, l.updated_at
		FROM leaves l
		LEFT JOIN employees e ON l.employee_id = e.id
		LEFT JOIN departments d ON e.department_id = d.id
		LEFT JOIN users u ON l.authorized_by = u.id
		WHERE l.id = ?`, id).
		Scan(&l.ID, &l.EmployeeID, &l.EmployeeName, &l.EmployeeNo, &l.Department,
			&l.Type, &l.StartDate, &l.EndDate, &l.Days, &reason, &l.Status,
			&authorizedBy, &l.AuthorizerName, &notes, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if reason.Valid {
		l.Reason = reason.String
	}
	if authorizedBy.Valid {
		l.AuthorizedBy = authorizedBy.String
	}
	if notes.Valid {
		l.Notes = notes.String
	}
	return l, nil
}

func (s *SQLiteStore) listLeavesQuery(ctx context.Context, where string, args ...interface{}) ([]*employees.Leave, error) {
	query := `SELECT l.id, l.employee_id,
		COALESCE(e.first_name||' '||e.last_name,'') as employee_name,
		COALESCE(e.employee_no,'') as employee_no,
		COALESCE(d.name,'') as department,
		l.type, l.start_date, l.end_date, l.days, l.reason, l.status,
		l.authorized_by, COALESCE(u.full_name,'') as authorizer_name,
		l.notes, l.created_at, l.updated_at
	FROM leaves l
	LEFT JOIN employees e ON l.employee_id = e.id
	LEFT JOIN departments d ON e.department_id = d.id
	LEFT JOIN users u ON l.authorized_by = u.id` + where + ` ORDER BY l.start_date DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*employees.Leave
	for rows.Next() {
		l := &employees.Leave{}
		var authorizedBy, notes, reason sql.NullString
		err := rows.Scan(&l.ID, &l.EmployeeID, &l.EmployeeName, &l.EmployeeNo, &l.Department,
			&l.Type, &l.StartDate, &l.EndDate, &l.Days, &reason, &l.Status,
			&authorizedBy, &l.AuthorizerName, &notes, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if reason.Valid {
			l.Reason = reason.String
		}
		if authorizedBy.Valid {
			l.AuthorizedBy = authorizedBy.String
		}
		if notes.Valid {
			l.Notes = notes.String
		}
		list = append(list, l)
	}
	return list, nil
}

func (s *SQLiteStore) ListLeaves(ctx context.Context) ([]*employees.Leave, error) {
	return s.listLeavesQuery(ctx, "")
}

func (s *SQLiteStore) ListLeavesByEmployee(ctx context.Context, employeeID string) ([]*employees.Leave, error) {
	return s.listLeavesQuery(ctx, " WHERE l.employee_id = ?", employeeID)
}

func (s *SQLiteStore) UpdateLeave(ctx context.Context, l *employees.Leave) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE leaves SET type = ?, start_date = ?, end_date = ?, days = ?,
			reason = ?, status = ?, authorized_by = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		l.Type, l.StartDate, l.EndDate, l.Days, l.Reason, l.Status, l.AuthorizedBy, l.Notes, l.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) DeleteLeave(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM leaves WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
