# Go Avanzado: Arquitectura de Eventos y CQRS

Este proyecto es una implementación de ejemplo de una arquitectura basada en eventos y CQRS (Command Query Responsibility Segregation) utilizando Go. El sistema maneja feeds de contenido, permitiendo crear, listar y buscar feeds de manera eficiente.

## Arquitectura

El proyecto sigue el patrón CQRS con microservicios desacoplados:

- **Command Side**: Maneja operaciones de escritura (crear feeds)
- **Query Side**: Maneja operaciones de lectura (listar y buscar feeds)
- **Event-Driven**: Comunicación asíncrona entre servicios mediante eventos
- **Real-time Push**: Notificaciones en tiempo real vía WebSockets

### Componentes Principales

- **Feed Service**: Servicio de comandos que crea nuevos feeds
- **Query Service**: Servicio de consultas que lista y busca feeds
- **Pusher Service**: Servicio de notificaciones en tiempo real
- **NATS**: Mensajería para eventos
- **PostgreSQL**: Base de datos principal para almacenamiento
- **Elasticsearch**: Motor de búsqueda para consultas eficientes
- **Nginx**: Proxy reverso para los servicios

## Flujo de Datos

1. **Creación de Feed**:
   - Cliente → Feed Service (POST /feeds)
   - Feed Service guarda en PostgreSQL
   - Feed Service publica evento `created_feed` en NATS

2. **Indexación**:
   - Query Service escucha evento `created_feed`
   - Query Service indexa feed en Elasticsearch

3. **Consultas**:
   - Cliente → Query Service (GET /feeds) → PostgreSQL
   - Cliente → Query Service (GET /search) → Elasticsearch

4. **Notificaciones en Tiempo Real**:
   - Pusher Service escucha evento `created_feed`
   - Pusher Service broadcast a clientes conectados vía WebSocket

## Tecnologías Utilizadas

- **Go 1.23**: Lenguaje principal
- **NATS Streaming**: Mensajería de eventos
- **PostgreSQL**: Base de datos relacional
- **Elasticsearch**: Motor de búsqueda
- **Gorilla Mux**: Router HTTP
- **Gorilla WebSocket**: WebSockets
- **Docker & Docker Compose**: Contenedorización

## Cómo Ejecutar

### Prerrequisitos

- Docker y Docker Compose instalados

### Pasos

1. Clona el repositorio
2. Ejecuta `docker-compose up --build`
3. Los servicios estarán disponibles en:
   - Feed Service: http://localhost:8080 (a través de Nginx)
   - Query Service: http://localhost:8080
   - Pusher Service: http://localhost:8080

## API Endpoints

### Feed Service
- `POST /feeds` - Crear un nuevo feed
  ```json
  {
    "title": "Título del feed",
    "description": "Descripción del feed"
  }
  ```

### Query Service
- `GET /feeds` - Listar todos los feeds
- `GET /search?q=query` - Buscar feeds
- `POST /reindex` - Reindexar todos los feeds en Elasticsearch
- `GET /health` - Verificar estado del servicio

### Pusher Service
- `GET /ws` - Conectar vía WebSocket para notificaciones en tiempo real

## Aplicaciones Posibles

Esta arquitectura es ideal para:

1. **Sistemas de Contenido**: Blogs, noticias, redes sociales
2. **E-commerce**: Catálogos de productos con búsqueda avanzada
3. **Plataformas de Streaming**: Gestión de metadatos de contenido
4. **Sistemas de Monitoreo**: Logs y eventos con búsqueda en tiempo real
5. **Aplicaciones IoT**: Procesamiento de datos de sensores
6. **Sistemas Financieros**: Transacciones y consultas separadas
7. **Plataformas Educativas**: Cursos y materiales con búsqueda
8. **Sistemas de Salud**: Registros médicos con consultas especializadas

## Beneficios de la Arquitectura

- **Escalabilidad**: Servicios independientes pueden escalar según necesidades
- **Rendimiento**: Consultas optimizadas con Elasticsearch
- **Tiempo Real**: Notificaciones instantáneas vía WebSockets
- **Mantenibilidad**: Separación clara de responsabilidades
- **Resiliencia**: Servicios desacoplados reducen puntos de falla
- **Flexibilidad**: Fácil agregar nuevos tipos de eventos y consultas