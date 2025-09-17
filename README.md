# goecommerce

[![Build Status](https://img.shields.io/github/actions/workflow/status/farhanalifianto/goecommerce/main.yml?branch=main)]()

## Table of Contents

- [Description](#description)
- [Features](#features)
- [Tech Stack / Key Dependencies](#tech-stack--key-dependencies)
- [File Structure Overview](#file-structure-overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage / Getting Started](#usage--getting-started)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)
- [Author/Acknowledgements](#authoracknowledgements)
- [Contact](#contact)

## Description

This repository contains the codebase for a Go-based e-commerce application. It includes multiple services such as user, product, and transaction management. It uses Docker Compose for orchestration.


## Features

- User management service
- Product management service
- Transaction management service
- Docker Compose orchestration for easy deployment
- Nginx configuration for reverse proxy and load balancing

## Tech Stack / Key Dependencies

- Go
- Docker
- Docker Compose
- Nginx

## File Structure Overview

```text
.

├── nginx
│   └── nginx.conf
├── product-service
│   ├── controller
│   │    └── product-controller.go
│   ├── routes
│   │    └── product-routes.go
│   ├── middleware
│   │    └── auth.go
│   ├── model
│   │    └── product.go
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── transaction-service
│   ├── controller
│   │    └── transaction-controller.go
│   ├── routes
│   │    └── transaction-routes.go
│   ├── middleware
│   │    └── auth.go
│   ├── model
│   │    └── transaction.go
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
└── user-service
│   ├── controller
│   │    └── user-controller.go
│   ├── routes
│   │    └── user-routes.go
│   ├── middleware
│   │    └── auth.go
│   ├── model
│   │    └── user.go
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
└── docker-compose.yml
```

## Prerequisites

- Docker
- Docker Compose

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/farhanalifianto/goecommerce.git
   cd goecommerce
   ```

2.  Start the services using Docker Compose:
    ```bash
    docker-compose up -d
    ```

## Usage / Getting Started

1. Access the application through your browser or API client.
2.  The specific endpoints and functionality depend on the individual services (user-service, product-service, transaction-service).



## Configuration

The application is configured using environment variables.  Refer to the individual service directories (user-service, product-service, transaction-service) for service-specific configuration instructions.



## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License


No license was explicitly found. All rights reserved.

## Author/Acknowledgements



## Contact

Your Name - [https://github.com/farhanalifianto/goecommerce](https://github.com/farhanalifianto/goecommerce) - email@example.com
