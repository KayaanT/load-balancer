package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

// Server represents a backend server
type Server struct {
	URL              string
	Healthy          bool
	ActiveConnections int
	TotalRequests    int
	Mutex            sync.Mutex
}

// LoadBalancer manages a pool of servers
type LoadBalancer struct {
	Servers []*Server
	Mutex   sync.Mutex
}

// Config holds the load balancer configuration
type Config struct {
	ListenPort          string   `json:"listenPort"`
	HealthCheckInterval string   `json:"healthCheckInterval"`
	Servers             []string `json:"servers"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(file string) (Config, error) {
	var config Config
	f, err := os.Open(file)
	if err != nil {
		return config, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)
	return config, err
}

// GetLeastConnectedServer returns the server with the least active connections
func (lb *LoadBalancer) GetLeastConnectedServer() *Server {
	lb.Mutex.Lock()
	defer lb.Mutex.Unlock()

	var leastConnectedServer *Server
	leastConnections := -1

	for _, server := range lb.Servers {
		server.Mutex.Lock()
		if !server.Healthy {
			server.Mutex.Unlock()
			continue
		}

		if leastConnections == -1 || server.ActiveConnections < leastConnections {
			leastConnections = server.ActiveConnections
			leastConnectedServer = server
		}
		server.Mutex.Unlock()
	}

	return leastConnectedServer
}

// HealthCheck periodically checks the health of all servers
func (lb *LoadBalancer) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		for _, server := range lb.Servers {
			go func(s *Server) {
				client := http.Client{
					Timeout: 5 * time.Second,
				}
				resp, err := client.Get(s.URL + "/health")
				
				s.Mutex.Lock()
				s.Healthy = err == nil && resp != nil && resp.StatusCode == http.StatusOK
				if resp != nil {
					resp.Body.Close()
				}
				s.Mutex.Unlock()
				
				if err != nil {
					log.Printf("Health check failed for %s: %s", s.URL, err)
				}
			}(server)
		}
	}
}

// GetStats returns statistics about all servers
func (lb *LoadBalancer) GetStats() []map[string]interface{} {
	lb.Mutex.Lock()
	defer lb.Mutex.Unlock()

	var stats []map[string]interface{}
	for _, server := range lb.Servers {
		server.Mutex.Lock()
		serverStats := map[string]interface{}{
			"url":               server.URL,
			"healthy":           server.Healthy,
			"activeConnections": server.ActiveConnections,
			"totalRequests":     server.TotalRequests,
		}
		server.Mutex.Unlock()
		stats = append(stats, serverStats)
	}

	return stats
}

// ServeHTTP handles incoming requests and forwards them to backend servers
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle API requests for the dashboard
	if r.URL.Path == "/api/server-stats" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lb.GetStats())
		return
	}
	
	// Serve the dashboard
	if r.URL.Path == "/" || r.URL.Path == "/dashboard" {
		http.ServeFile(w, r, "dashboard.html")
		return
	}

	// Handle load balancing for other requests
	server := lb.GetLeastConnectedServer()
	if server == nil {
		http.Error(w, "No healthy servers available", http.StatusServiceUnavailable)
		return
	}

	// Parse the URL
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		http.Error(w, "Error parsing server URL", http.StatusInternalServerError)
		return
	}

	// Increment connection count
	server.Mutex.Lock()
	server.ActiveConnections++
	server.TotalRequests++
	server.Mutex.Unlock()

	// Create a reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// Define the director func
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", targetURL.Host)
	}

	// Modify the response to handle connection tracking
	proxy.ModifyResponse = func(resp *http.Response) error {
		time.Sleep(5 * time.Second)
		server.Mutex.Lock()
		server.ActiveConnections--
		server.Mutex.Unlock()
		return nil
	}

	// Handle errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		server.Mutex.Lock()
		server.ActiveConnections--
		server.Healthy = false // Mark as unhealthy if proxy fails
		server.Mutex.Unlock()
		
		http.Error(w, "Error proxying request: "+err.Error(), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

func main() {
	// Load configuration
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Printf("Error loading config: %s. Using default configuration.", err)
		config = Config{
			ListenPort:          ":8080",
			HealthCheckInterval: "10s",
			Servers:             []string{"http://localhost:8081", "http://localhost:8082"},
		}
	}

	// Initialize load balancer
	lb := &LoadBalancer{}
	
	// Initialize servers
	for _, serverURL := range config.Servers {
		lb.Servers = append(lb.Servers, &Server{
			URL:     serverURL,
			Healthy: true,
		})
	}

	// Start health checks
	interval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		interval = 10 * time.Second
	}
	go lb.HealthCheck(interval)

	// Create dashboard.html file
	createDashboardFile()

	// Start the server
	fmt.Printf("Load balancer starting on port %s\n", config.ListenPort)
	log.Fatal(http.ListenAndServe(config.ListenPort, lb))
}

// createDashboardFile creates the HTML file for the dashboard
func createDashboardFile() {
	dashboard := `<!DOCTYPE html>
<html>
<head>
    <title>Load Balancer Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .chart-container { height: 400px; margin-bottom: 20px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        .healthy { color: green; }
        .unhealthy { color: red; }
        .refresh-rate { margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Load Balancer Dashboard</h1>
        
        <div class="refresh-rate">
            Refresh rate: 
            <select id="refresh-rate">
                <option value="1000">1 second</option>
                <option value="5000" selected>5 seconds</option>
                <option value="10000">10 seconds</option>
                <option value="30000">30 seconds</option>
            </select>
        </div>
        
        <div class="chart-container">
            <canvas id="loadChart"></canvas>
        </div>
        
        <h2>Server Status</h2>
        <table id="serverTable">
            <thead>
                <tr>
                    <th>Server URL</th>
                    <th>Status</th>
                    <th>Active Connections</th>
                    <th>Total Requests</th>
                </tr>
            </thead>
            <tbody id="serverTableBody">
                <!-- Server data will be inserted here -->
            </tbody>
        </table>
    </div>

    <script>
        let chart;
        let refreshInterval;
        
        // Initialize the chart
        function initChart() {
            const ctx = document.getElementById('loadChart').getContext('2d');
            chart = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: [],
                    datasets: [{
                        label: 'Active Connections',
                        backgroundColor: 'rgba(75, 192, 192, 0.6)',
                        data: []
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        }
        
        // Fetch server stats and update the UI
        async function updateStats() {
            try {
                const response = await fetch('/api/server-stats');
                const data = await response.json();
                
                // Update chart
                chart.data.labels = data.map(server => server.url);
                chart.data.datasets[0].data = data.map(server => server.activeConnections);
                chart.update();
                
                // Update table
                const tableBody = document.getElementById('serverTableBody');
                tableBody.innerHTML = '';
                
                data.forEach(server => {
                    const row = document.createElement('tr');
                    
                    const urlCell = document.createElement('td');
                    urlCell.textContent = server.url;
                    
                    const statusCell = document.createElement('td');
                    statusCell.textContent = server.healthy ? 'Healthy' : 'Unhealthy';
                    statusCell.className = server.healthy ? 'healthy' : 'unhealthy';
                    
                    const connectionsCell = document.createElement('td');
                    connectionsCell.textContent = server.activeConnections;
                    
                    const requestsCell = document.createElement('td');
                    requestsCell.textContent = server.totalRequests;
                    
                    row.appendChild(urlCell);
                    row.appendChild(statusCell);
                    row.appendChild(connectionsCell);
                    row.appendChild(requestsCell);
                    
                    tableBody.appendChild(row);
                });
            } catch (error) {
                console.error('Error fetching server stats:', error);
            }
        }
        
        // Set up refresh rate change handler
        document.getElementById('refresh-rate').addEventListener('change', function() {
            const rate = parseInt(this.value);
            clearInterval(refreshInterval);
            refreshInterval = setInterval(updateStats, rate);
        });
        
        // Initialize the dashboard
        document.addEventListener('DOMContentLoaded', function() {
            initChart();
            updateStats();
            refreshInterval = setInterval(updateStats, 5000); // Default refresh rate
        });
    </script>
</body>
</html>`;

	err := os.WriteFile("dashboard.html", []byte(dashboard), 0644)
	if err != nil {
		log.Printf("Error creating dashboard file: %s", err)
	}
}
