#!/bin/bash
set -e

# Function to process a SQL file
process_sql_file() {
  local file=$1
  local filename=$(basename "$file")

  # Check if it's a default file
  if [[ "$filename" == "default.sql" ]]; then
    echo "Loading default file $file into $POSTGRES_DB database..."
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=0 -f "$file"
    echo "Finished loading $file into $POSTGRES_DB database"
    return
  fi

  # Check if it's a database-specific file
  if [[ "$filename" =~ ^([^\.]+)\.sql$ && "$filename" != "default.sql" ]]; then
    local db_name="${BASH_REMATCH[1]}"
    echo "Creating database $db_name and loading $file..."
    # Create database if it doesn't exist
    psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $db_name WITH OWNER $POSTGRES_USER;" || true
    # Load the dump into the new database
    psql -U "$POSTGRES_USER" -d "$db_name" -v ON_ERROR_STOP=0 -f "$file"
    echo "Finished loading $file into $db_name database"
    return
  fi

}

# Function to process a gzipped SQL file
process_gzipped_sql_file() {
  local file=$1
  local filename=$(basename "$file")

  # Check if it's a default file
  if [[ "$filename" == "default.sql.gz" ]]; then
    echo "Loading default file $file into $POSTGRES_DB database..."
    gunzip -c "$file" | psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=0
    echo "Finished loading $file into $POSTGRES_DB database"
    return
  fi

  # Check if it's a database-specific file
  if [[ "$filename" =~ ^([^\.]+)\.sql\.gz$ && "$filename" != "default.sql.gz" ]]; then
    local db_name="${BASH_REMATCH[1]}"
    echo "Creating database $db_name and loading $file..."
    # Create database if it doesn't exist
    psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $db_name WITH OWNER $POSTGRES_USER;" || true
    # Load the dump into the new database
    gunzip -c "$file" | psql -U "$POSTGRES_USER" -d "$db_name" -v ON_ERROR_STOP=0
    echo "Finished loading $file into $db_name database"
    return
  fi

}

# Process plain SQL files
echo "Processing .sql files..."
for file in /var/lib/postgresql/pg-backup/*.sql; do
  if [ -f "$file" ]; then
    process_sql_file "$file"
  fi
done

# Process gzipped SQL files
echo "Processing .sql.gz files..."
for file in /var/lib/postgresql/pg-backup/*.sql.gz; do
  if [ -f "$file" ]; then
    process_gzipped_sql_file "$file"
  fi
done

echo "All SQL files have been loaded."
