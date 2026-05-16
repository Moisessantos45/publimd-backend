# Publimd Backend API

Publimd es una plataforma web innovadora para la creación y publicación de posts tipo blog. Su principal objetivo es permitir a los usuarios escribir contenido de forma rápida y sencilla utilizando Markdown (MD), mientras facilita la colaboración en equipo en tiempo real. 

Este repositorio contiene la **API Backend**, encargada de proveer toda la lógica de negocio, almacenamiento, gestión de búsqueda inteligente y funcionalidades en tiempo real para la plataforma.

## Características Principales

- **API RESTful de Alto Rendimiento:** Desarrollada con Gin Framework para responder con latencias mínimas.
- **Búsqueda Avanzada e Inteligente:** 
  - **Búsqueda Semántica:** Uso de la extensión **pgvector** (`vector(768)`) para almacenar embeddings y realizar búsquedas por similitud de contexto.
  - **Full-Text Search:** Optimizado mediante `tsvector` con configuración de idioma personalizada (`spanish_unaccent`) y diccionarios *stem*, asignando pesos a títulos, categorías, etiquetas y contenido.
  - **Búsqueda Difusa (Fuzzy Search):** Implementada mediante la extensión `pg_trgm` y un campo dinámico (`fuzzy_short`) manejado a través de triggers y funciones SQL para tolerar errores tipográficos y búsquedas parciales eficientes.
- **Edición Colaborativa en Tiempo Real:** Implementación de **WebSockets** en Go que permite a múltiples autores trabajar y sincronizar cambios en el mismo post de forma conjunta y sin latencia perceptible.
- **Sistema de Caché Optimizada:** Integración profunda con **Redis** para mantener en caché los resultados más frecuentes y aliviar la carga de la base de datos principal.

## Tecnologías y Herramientas

- **[Go (Golang)](https://go.dev/):** Lenguaje principal, elegido por su increíble manejo de concurrencia y velocidad.
- **[Gin Web Framework](https://gin-gonic.com/):** Router y framework HTTP moderno y eficiente.
- **[PostgreSQL](https://www.postgresql.org/):** Base de datos relacional robusta.
- **[GORM](https://gorm.io/):** ORM (Object Relational Mapper) orientado a desarrolladores para gestionar modelos e interactuar con PostgreSQL.
- **[Redis](https://redis.io/):** Motor en memoria clave-valor para almacenamiento de sesiones y caché de consultas.

## Arquitectura de la Base de Datos para Búsquedas

La base de datos aprovecha al máximo las capacidades avanzadas de PostgreSQL a través de extensiones:

- **Vector:** `CREATE EXTENSION IF NOT EXISTS vector;` (para embeddings vectoriales).
- **Trigramas:** `CREATE EXTENSION IF NOT EXISTS pg_trgm;` (índices GIN trigramas para comparaciones LIKE rápidas).
- **Unaccent:** `CREATE EXTENSION IF NOT EXISTS unaccent;` (para normalizar acentos en búsquedas Full-Text).

Toda la complejidad del esquema se gestiona automáticamente para ofrecer los resultados de búsqueda más precisos posibles a la aplicación cliente.

## Frontend y Demo

- **Repositorio del Frontend:** [Moisessantos45/publimd_frontend](https://github.com/Moisessantos45/publimd_frontend)
- **Sitio Web en Vivo (Demo):** [https://publimd.mmabitec.me/](https://publimd.mmabitec.me/)
