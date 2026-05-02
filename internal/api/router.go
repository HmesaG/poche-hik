package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"ponches/internal/auth"
	"ponches/internal/config"
	"ponches/internal/discovery"
	"ponches/internal/middleware"
	"ponches/internal/realtime"
	"ponches/internal/store"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Server struct {
	Router     *chi.Mux
	Config     *config.Config
	Store      store.Repository
	Hub        *realtime.Hub
	JWTService *auth.JWTService
	mu         sync.RWMutex // Protects config updates

	// Rate limiters
	apiLimiter  *middleware.RateLimiter
	authLimiter *middleware.AuthRateLimiter
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return sameOrigin(r)
	},
}

func NewServer(cfg *config.Config, s store.Repository, h *realtime.Hub) *Server {
	srv := &Server{
		Router:      chi.NewRouter(),
		Config:      cfg,
		Store:       s,
		Hub:         h,
		JWTService:  auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiration),
		apiLimiter:  middleware.NewRateLimiter(10, 20),               // 10 req/s, burst 20
		authLimiter: middleware.NewAuthRateLimiter(5, 5*time.Minute), // 5 attempts per 5 min
	}

	srv.Router.Use(chimiddleware.Logger)
	srv.Router.Use(chimiddleware.Recoverer)
	srv.Router.Use(chimiddleware.RealIP)
	srv.Router.Use(chimiddleware.Timeout(60 * time.Second))
	srv.Router.Use(secureHeaders)

	srv.routes()
	return srv
}

func (s *Server) routes() {
	s.Router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	s.Router.Get("/manifest.webmanifest", serveWebAsset("manifest.webmanifest", "application/manifest+json; charset=utf-8", "no-cache"))
	s.Router.Get("/service-worker.js", serveWebAsset("service-worker.js", "application/javascript; charset=utf-8", "no-cache"))
	s.Router.Get("/offline.html", serveWebAsset("offline.html", "text/html; charset=utf-8", "no-cache"))
	s.Router.Get("/directorio", serveWebAsset("directorio.html", "text/html; charset=utf-8", "no-cache"))
	s.Router.Get("/directory", serveWebAsset("directorio.html", "text/html; charset=utf-8", "no-cache"))
	s.Router.Get("/icons/icon.svg", serveWebAsset(filepath.Join("assets", "icons", "icon.svg"), "image/svg+xml", "public, max-age=604800"))
	s.Router.Get("/icons/icon-maskable.svg", serveWebAsset(filepath.Join("assets", "icons", "icon-maskable.svg"), "image/svg+xml", "public, max-age=604800"))

	// Serve uploaded files
	s.Router.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Join("web", "assets", "uploads")))))

	s.Router.Get("/ws", s.handleWS)

	// Public API routes (no auth required)
	s.Router.Route("/api/public", func(r chi.Router) {
		r.Use(s.authLimiter.Middleware) // Strict rate limiting for auth
		r.Post("/auth/login", s.handleLogin)
		r.Get("/directory", s.handlePublicDirectory)
		r.Get("/directory/{employeeNo}/contact.vcf", s.handlePublicDirectoryContact)
		r.Post("/callback/hikvision", s.handleHikvisionCallback)
	})

	// Protected API routes (auth required)
	s.Router.Route("/api", func(r chi.Router) {
		// Apply rate limiting
		r.Use(s.apiLimiter.Middleware)

		// Apply JWT middleware
		r.Use(s.JWTService.Middleware)

		// Auth routes
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// Users management (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/users", s.handleRegisterUser)
			r.Get("/users", s.handleListUsers)
			r.Get("/users/{id}", s.handleGetUser)
			r.Put("/users/{id}", s.handleUpdateUser)
			r.Delete("/users/{id}", s.handleDeleteUser)
			r.Get("/audit-logs", s.handleListAuditLogs)
		})

		// Discovery (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Get("/devices/configured", s.handleListManagedDevices)
			r.Post("/devices/configured", s.handleCreateManagedDevice)
			r.Put("/devices/configured/{id}", s.handleUpdateManagedDevice)
			r.Delete("/devices/configured/{id}", s.handleDeleteManagedDevice)
			r.Post("/devices/configured/{id}/default", s.handleSetManagedDeviceDefault)
			r.Post("/devices/configured/{id}/sync", s.handleSyncEmployeesToDevice)
			r.Post("/devices/import-users", s.handleImportEmployeesFromDevices)
			r.Post("/devices/import-photos", s.handleImportAllFaces)
			r.Post("/devices/read-events", s.handleReadRecentEvents)
			r.Post("/devices/configured/{id}/sync-one/{employeeNo}", s.handleSyncOneEmployeeToDevice)
			r.Post("/devices/configured/{id}/sync-time", s.handleSyncDeviceTime)
			r.Post("/devices/configured/{id}/setup-alarm-host", s.handleSetupDeviceAlarmHost)
			r.Delete("/devices/configured/{id}/sync-one/{employeeNo}", s.handleRevokeEmployeeFromDevice)
			r.Get("/devices/configured/{id}/logs", s.handleGetDeviceLogs)
			r.Get("/devices/logs", s.handleGetDeviceLogs)
			r.Get("/discovery/discover", s.handleDiscoverDevices)
			r.Get("/discovery/refresh", s.handleRefreshDevices)
			r.Get("/config/network", s.handleGetNetworkConfig)
			r.Post("/config/network", s.handleUpdateNetworkConfig)
		})

		// Employees (admin, manager)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "manager"))
			r.Post("/employees", s.handleCreateEmployee)
			r.Get("/employees/{id}", s.handleGetEmployee)
			r.Put("/employees/{id}", s.handleUpdateEmployee)
			r.Delete("/employees/{id}", s.handleDeleteEmployee)
			r.Put("/employees/{employeeNo}/photo", s.handleUploadEmployeePhoto)
			r.Delete("/employees/{employeeNo}/photo", s.handleDeleteEmployeePhoto)
			r.Post("/employees/{employeeNo}/face", s.handleRegisterFace)
			r.Post("/employees/{employeeNo}/face/import", s.handleImportFace)
			r.Delete("/employees/{employeeNo}/face", s.handleDeleteFace)
			r.Get("/employees/{employeeNo}/face/status", s.handleFaceStatus)
			r.Get("/employees/faces/list", s.handleListFaces)
		})
		// List employees - viewer can also read
		r.Get("/employees", s.handleListEmployees)

		// Organization - Departments (admin, manager for write; all for read)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "manager"))
			r.Post("/departments", s.handleCreateDepartment)
			r.Get("/departments/{id}", s.handleGetDepartment)
			r.Put("/departments/{id}", s.handleUpdateDepartment)
			r.Delete("/departments/{id}", s.handleDeleteDepartment)
		})
		r.Get("/departments", s.handleListDepartments)

		// Organization - Positions (admin, manager for write; all for read)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "manager"))
			r.Post("/positions", s.handleCreatePosition)
			r.Get("/positions/{id}", s.handleGetPosition)
			r.Put("/positions/{id}", s.handleUpdatePosition)
			r.Delete("/positions/{id}", s.handleDeletePosition)
		})
		r.Get("/positions", s.handleListPositions)

		// Attendance routes
		r.Get("/attendance/events", s.handleGetEvents)
		r.Get("/attendance/stats", s.handleGetDashboardStats)
		r.Get("/attendance/recent", s.handleGetRecentActivity)
		r.Post("/attendance/process", s.handleProcessAttendance)
		r.Get("/attendance/daily", s.handleGetDailyAttendance)
		r.Get("/attendance/stats", s.handleGetStats)

		// Reports (all authenticated users)
		r.Get("/reports/daily", s.handleReportDaily)
		r.Get("/reports/payroll", s.handleReportPayroll)
		r.Get("/reports/late", s.handleReportLate)
		r.Get("/reports/kpis", s.handleReportKPIs)
		r.Get("/reports/attendance", s.handleReportAttendancePeriod)
		r.Get("/reports/attendance/data", s.handleReportAttendanceData)

		r.Get("/leaves", s.handleListLeaves)
		r.Get("/leaves/{id}", s.handleGetLeave)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "manager"))
			r.Post("/leaves", s.handleCreateLeave)
			r.Put("/leaves/{id}", s.handleUpdateLeave)
			r.Delete("/leaves/{id}", s.handleDeleteLeave)
		})

		// Notify employee (build WhatsApp/email message)
		r.Post("/notify/employee", s.handleNotifyEmployee)

		// Config (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Get("/config", s.handleGetConfig)
			r.Post("/config", s.handleUpdateConfig)
			r.Post("/config/rnc-lookup", s.handleProxyRNC)
		})

		// LDAP (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/ldap/test", s.handleTestLDAP)
			r.Post("/ldap/sync", s.handleSyncLDAP)
		})

		// Holidays (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/holidays", s.handleCreateHoliday)
			r.Put("/holidays/{id}", s.handleUpdateHoliday)
			r.Delete("/holidays/{id}", s.handleDeleteHoliday)
		})
		r.Get("/holidays", s.handleListHolidays)
		r.Get("/holidays/{id}", s.handleGetHoliday)

		// Travel Allowance Rates (admin only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/travel-rates", s.handleCreateTravelRate)
			r.Put("/travel-rates/{id}", s.handleUpdateTravelRate)
			r.Delete("/travel-rates/{id}", s.handleDeleteTravelRate)
		})
		r.Get("/travel-rates", s.handleListTravelRates)

		// Travel Allowances (admin, manager for write; all for read)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "manager"))
			r.Post("/travel-allowances", s.handleCreateTravelAllowance)
			r.Put("/travel-allowances/{id}", s.handleUpdateTravelAllowance)
			r.Delete("/travel-allowances/{id}", s.handleDeleteTravelAllowance)
			r.Post("/travel-allowances/{id}/approve", s.handleApproveTravelAllowance)
			r.Post("/travel-allowances/{id}/reject", s.handleRejectTravelAllowance)
		})
		r.Get("/travel-allowances", s.handleListTravelAllowances)
		r.Get("/travel-allowances/{id}", s.handleGetTravelAllowance)
		r.Get("/travel-allowances/{id}/pdf", s.handleTravelAllowancePDF)
	})

	// Static Files - Must be last to not interfere with API routes
	s.Router.Handle("/*", http.FileServer(http.Dir("./web")))
}

// Handlers
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	devices, err := discovery.Discover(s.Config.SADPTimeoutSeconds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	emps, err := s.Store.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.markEmployeesAdminStatus(r.Context(), emps); err != nil {
		http.Error(w, "failed to resolve employee roles", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(emps)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := auth.ExtractTokenFromRequest(r)
	if token == "" {
		http.Error(w, "missing websocket token", http.StatusUnauthorized)
		return
	}
	if _, err := s.JWTService.ValidateToken(token); err != nil {
		http.Error(w, "invalid websocket token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket")
		return
	}
	s.Hub.Register(conn)
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	return strings.EqualFold(parsed.Host, r.Host)
}

func serveWebAsset(name, contentType, cacheControl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		if cacheControl != "" {
			w.Header().Set("Cache-Control", cacheControl)
		}
		http.ServeFile(w, r, filepath.Join("web", name))
	}
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.Store.GetEvents(r.Context(), store.EventFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(events)
}
