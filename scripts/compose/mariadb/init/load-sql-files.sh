#!/bin/bash
set -e

# Determine which client command to use (mysql or mariadb)
if command -v mysql &> /dev/null; then
    DB_CLIENT="mysql"
elif command -v mariadb &> /dev/null; then
    DB_CLIENT="mariadb"
else
    echo "Error: Neither mysql nor mariadb client found in the container"
    exit 1
fi

echo "Using database client: $DB_CLIENT"

# Function to process a SQL file
process_sql_file() {
  local file=$1
  local filename=$(basename "$file")

  # Check if it's a default file
  if [[ "$filename" == "default.sql" ]]; then
    echo "Loading default file $file into $MYSQL_DATABASE database..."
    $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" < "$file"
    echo "Finished loading $file into $MYSQL_DATABASE database"
    return
  fi

  # Check if it's a database-specific file
  if [[ "$filename" =~ ^([^\.]+)\.sql$ && "$filename" != "default.sql" ]]; then
    local db_name="${BASH_REMATCH[1]}"
    echo "Creating database $db_name and loading $file..."
    # Create database if it doesn't exist
    $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS $db_name;" || true
    # Grant privileges to the user
    $DB_CLIENT -u root -p"$MYSQL_ROOT_PASSWORD" -e "GRANT ALL PRIVILEGES ON $db_name.* TO '$MYSQL_USER'@'%';" || true
    $DB_CLIENT -u root -p"$MYSQL_ROOT_PASSWORD" -e "FLUSH PRIVILEGES;" || true
    # Load the dump into the new database
    $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$db_name" < "$file"
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
    echo "Loading default file $file into $MYSQL_DATABASE database..."
    gunzip -c "$file" | $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE"
    echo "Finished loading $file into $MYSQL_DATABASE database"
    return
  fi

  # Check if it's a database-specific file
  if [[ "$filename" =~ ^([^\.]+)\.sql\.gz$ && "$filename" != "default.sql.gz" ]]; then
    local db_name="${BASH_REMATCH[1]}"
    echo "Creating database $db_name and loading $file..."
    # Create database if it doesn't exist
    $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS $db_name;" || true
    # Grant privileges to the user
    $DB_CLIENT -u root -p"$MYSQL_ROOT_PASSWORD" -e "GRANT ALL PRIVILEGES ON $db_name.* TO '$MYSQL_USER'@'%';" || true
    $DB_CLIENT -u root -p"$MYSQL_ROOT_PASSWORD" -e "FLUSH PRIVILEGES;" || true
    # Load the dump into the new database
    gunzip -c "$file" | $DB_CLIENT -u "$MYSQL_USER" -p"$MYSQL_PASSWORD" "$db_name"
    echo "Finished loading $file into $db_name database"
    return
  fi
}

# Process plain SQL files
echo "Processing .sql files..."
for file in /var/lib/mysql/backup/*.sql; do
  if [ -f "$file" ]; then
    process_sql_file "$file"
  fi
done

# Process gzipped SQL files
echo "Processing .sql.gz files..."
for file in /var/lib/mysql/backup/*.sql.gz; do
  if [ -f "$file" ]; then
    process_gzipped_sql_file "$file"
  fi
done

echo "All SQL files have been loaded."
