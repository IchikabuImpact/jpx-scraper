# JPX Scraper

JPX Scraper is a Go application that scrapes stock data from JPX using Selenium and Docker.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker
- Docker Compose (optional, but recommended)

### Setup and Installation
```
Clone the repository

   bash
   git clone https://github.com/IchikabuImpact/jpx-scraper.git
   cd jpx-scraper
Build Docker images

bash
docker build -t jpx-scraper -f Dockerfile.go .
docker build -t selenium-hub-custom -f Dockerfile.selenium .
Running the Application
Start Selenium Hub and Node

bash
docker run -d -p 4445:4445 --name selenium-hub --network selenium-network selenium-hub-custom
docker run -d --network selenium-network --link selenium-hub:hub -e SE_EVENT_BUS_HOST=selenium-hub -e SE_EVENT_BUS_PUBLISH_PORT=4442 -e SE_EVENT_BUS_SUBSCRIBE_PORT=4443 selenium/node-chrome
Start JPX Scraper

bash
docker run -d -p 8082:8081 --network selenium-network --name jpx-scraper jpx-scraper
Accessing the Application
You can access the scraper by visiting the following URL:

bash
http://localhost:8082/scrape?ticker=1332
Replace 1332 with the desired stock ticker.

Apache Configuration (Optional)
If you are using Apache as a reverse proxy, make sure to configure it to forward requests to the JPX Scraper.

apache
<VirtualHost *:443>
    ServerName jpx.pinkgold.space
    DocumentRoot /var/www/jpx-scraper/public
    ErrorLog "/var/log/httpd/jpx_pinkgold_space_error_log"
    CustomLog "/var/log/httpd/jpx_pinkgold_space_access_log" combined
    <Directory "/var/www/jpx-scraper/public">
        Options -Indexes +FollowSymLinks
        AllowOverride all
        Require all granted
    </Directory>

    ProxyPass "/scrape" "http://localhost:8082/scrape"
    ProxyPassReverse "/scrape" "http://localhost:8082/scrape"

    SSLCertificateFile /etc/letsencrypt/live/jpx.pinkgold.space/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/jpx.pinkgold.space/privkey.pem
    Include /etc/letsencrypt/options-ssl-apache.conf
</VirtualHost>
Debugging and Troubleshooting
View Docker logs

docker logs jpx-scraper
docker logs selenium-hub
Check active ports

docker exec -it jpx-scraper /bin/sh
netstat -tuln
Install utilities inside Docker container (if needed)

docker exec -it jpx-scraper /bin/sh
apk update
apk add curl
Test scraper locally

curl http://localhost:8082/scrape?ticker=1332
Known Issues
Ensure that the ports 4445 (Selenium Hub) and 8082 (JPX Scraper) are not being used by other applications.
If the application fails to start, check the Docker logs for errors and ensure that all prerequisites are installed.
Contributing
Feel free to fork this project and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.

License
This project is licensed under the MIT License.

Ports Summary
Component	Host Port	Container Port	Description
Selenium Hub	4445	4445	Selenium Grid Hub
JPX Scraper	8082	8081	JPX Scraper API
Apache	443	N/A	HTTPS Proxy to JPX Scraper
Commands for Running and Debugging
Start Selenium Hub and Node:

docker run -d -p 4445:4445 --name selenium-hub --network selenium-network selenium-hub-custom
docker run -d --network selenium-network --link selenium-hub:hub -e SE_EVENT_BUS_HOST=selenium-hub -e SE_EVENT_BUS_PUBLISH_PORT=4442 -e SE_EVENT_BUS_SUBSCRIBE_PORT=4443 selenium/node-chrome
Start JPX Scraper:

docker run -d -p 8082:8081 --network selenium-network --name jpx-scraper jpx-scraper
View Docker logs:

docker logs jpx-scraper
docker logs selenium-hub
Check active ports:

docker exec -it jpx-scraper /bin/sh
netstat -tuln
Install utilities inside Docker container:

docker exec -it jpx-scraper /bin/sh
apk update
apk add curl
Test scraper locally:

curl http://localhost:8082/scrape?ticker=1332
```
