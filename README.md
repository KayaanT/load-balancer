# Custom Load Balancer in Go with Real-Time Visualization

This project implements a custom software-based load balancer in Go, complete with a real-time dashboard visualization. It dynamically distributes incoming HTTP requests across multiple backend servers using the **Least Connections** algorithm.

---

## High-Level Project Overview

1. **Backend Servers**: Simple Go HTTP servers that handle incoming requests and respond after a configurable delay.
2. **Custom Load Balancer**: A Go-based load balancer that forwards incoming HTTP requests to backend servers.
3. **Health Checks**: Periodically checks if backend servers are healthy and dynamically removes unhealthy servers from rotation.
4. **Visualization Dashboard**: A real-time dashboard built with HTML, JavaScript, and Chart.js to visualize server load and health status.
5. **AWS Infrastructure**: Deployment of backend servers and load balancer onto AWS EC2 instances.
6. **Nginx Reverse Proxy**: Nginx server set up as a reverse proxy in front of the Go load balancer to provide standard HTTP port access, SSL termination, and efficient static content serving.
7. **Load Testing**: Using tools like ApacheBench (`ab`) or Siege to simulate real-world traffic and verify correct load distribution.

---

## Technologies Used

| Technology               | Purpose                                        |
|---------------------------|---------------------------------------------------|
| Go                        | Backend & Load Balancer logic                   |
| Nginx                     | Reverse proxy & static content serving           |
| AWS EC2                   | Cloud hosting infrastructure                     |
| Systemd                   | Service management                               |
| Apache Bench (ab), curl   | Load testing                                     |
| Chart.js                  | Real-time visualization                          |
| HTML/CSS/JavaScript       | Dashboard frontend                               |

---

## Development Steps

### Step 1: Backend Servers Setup

- Write simple HTTP servers in Go.
- Implement `/health` endpoint for health checks.
- Include artificial delays for testing purposes.

### Step 2: Load Balancer Implementation

- Implemented in Go, using the Least Connections algorithm.
- Regularly performs health checks on backend servers.
- Provides API endpoints for real-time statistics visualization.

### Step 3: Visualization Dashboard

- HTML dashboard using Chart.js to visualize active connections per server.
- Fetches real-time data from `/api/server-stats` endpoint provided by the load balancer.

### Step 4: AWS Deployment

- Deploy backend servers and load balancer on separate EC2 instances.
- Configure security groups to allow necessary traffic (HTTP ports).

### Step 5: Nginx Reverse Proxy Setup

- Installed on the load balancer instance.
- Forwards HTTP requests from port 80 to the Go application running on port 8080.

### Step 6: Load Testing & Monitoring

- Use Apache Bench (`ab`) or Siege to simulate high traffic scenarios.
- Monitor real-time server loads through the visualization dashboard.

---

## How the Load Balancing Algorithm Works (**Least Connections Algorithm**)

The implemented algorithm is called "**Least Connections**," a dynamic load balancing method that distributes incoming requests based on the current number of active connections each backend server has:

- When a new request arrives, the load balancer checks all available and healthy backend servers.
- It selects the server currently handling the fewest active connections.
- The request is forwarded to this selected server, incrementing its active connection count.
- Once the request completes (and after an optional delay), the active connection count is decremented.

---
