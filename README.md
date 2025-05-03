#Project - Kube

A locally deplyable drive space with advanced intelligent file search and advanced compression bays (Refrigirator) and all time intelligent security protocols/

User Guide
System Requirements
•	Operating Systems: Windows 10/11, macOS 10.15+, or Linux
•	RAM: Minimum 4GB (8GB recommended)
•	Storage: 500MB for application, additional space for stored files
•	Network: Active LAN connection required for node discovery
Installation
Setup Development Environment
1.Install Golang programming language:
•	Download from golang.org
•	Verify installation: go version
2.	Install Wails:
•	Run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
•	Verify installation: wails version
3.	Clone the Repository:
4.	git clone https://github.com/yourusername/kube-app.git
5.	cd kube-app
Building the Application
•	Development Mode: wails dev
•	Production Build:
o	Windows: wails build -platform windows
o	macOS: wails build -platform darwin
o	Linux: wails build -platform linux


Using Kube
Initial Setup
1.	Launch the application
2.	The system will automatically discover other Kube nodes on your network
3.	Grant camera permissions when prompted (required for facial recognition security)
Network Visualization
•	View all connected nodes in the Network panel
•	Green nodes represent online systems
•	Yellow nodes indicate locked systems (user away)
•	Red nodes show offline systems
•	Click any online node to establish a connection
File Management
1.	Navigate to the Files tab
2.	Drag and drop files to upload them to the distributed storage
3.	Files will be automatically compressed and distributed across nodes
4.	Download files by selecting them and clicking "Download"
Security Features
•	Auto-Lock: The system uses facial recognition to detect when you leave your desk
•	The node status automatically changes to "locked" and propagates to all network nodes
•	Return to your desk to unlock (facial recognition will verify your identity)
Troubleshooting
•	If nodes aren't discovered, check your firewall settings
•	For connection issues, ensure all nodes are on the same subnet
