# PostgreSQL Database Initialization

This directory contains scripts that are automatically executed when the PostgreSQL Docker container is initialized. The scripts are used to load SQL backup files into PostgreSQL databases.

## How it Works

The main script in this directory is `load-sql-files.sh`, which is designed to:

1. Process SQL backup files from the `/var/lib/postgresql/pg-backup` directory (mounted from `./pg/backup` in the host)
2. Create databases as needed and load the SQL dumps into them

### Integration with Docker

In the `docker-compose.yml` file, this directory is mounted to the PostgreSQL container's `/docker-entrypoint-initdb.d` directory:

```yaml
volumes:
  - ./pg/init:/docker-entrypoint-initdb.d
  - ./pg/backup:/var/lib/postgresql/pg-backup
```

The PostgreSQL Docker image automatically executes any scripts in the `/docker-entrypoint-initdb.d` directory during container initialization.

### Script Behavior

The `load-sql-files.sh` script processes two types of SQL files:

1. Plain SQL files (`.sql`)
2. Gzipped SQL files (`.sql.gz`)

For each file, it follows these rules:

#### Default Database Files

If a file is named `default.sql` or `default.sql.gz`, it will be loaded into the default database specified by the `POSTGRES_DB` environment variable (set to `twc_db` in the docker-compose.yml).

Example:
```
default.sql.gz → loaded into twc_db database
```

#### Database-Specific Files

If a file has any other name with a `.sql` or `.sql.gz` extension, the script will:
1. Create a new database with the same name as the file (without the extension)
2. Load the SQL dump into that database

Example:
```
executive_db.sql.gz → creates a database named "executive_db" and loads the SQL into it
```

### Error Handling

The script uses `ON_ERROR_STOP=0` when executing SQL, which means it will continue processing even if some SQL statements fail. This is useful for handling dumps that might contain statements that depend on specific database configurations.

## Usage

To use this initialization system:

1. Place your SQL dump files in the `pg/backup` directory
2. Name your files according to the rules above
3. Start the Docker containers with `docker-compose up`

The script will automatically process all SQL files in the `pg/backup` directory and load them into the appropriate databases.
