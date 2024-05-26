# Check if a title was provided
if [ -z "$1" ]; then
    echo "Usage: $0 <title>"
    exit 1
fi

# Define variables
title=$1
version=$(date +%s)
extension="sql"
migrations_dir="migrations"

# Create the migrations directory if it doesn't exist
if [ ! -d "$migrations_dir" ]; then
    mkdir "$migrations_dir"
fi

# Create migration files
up_file="${migrations_dir}/${version}_${title}.up.${extension}"
down_file="${migrations_dir}/${version}_${title}.down.${extension}"

cat >"$up_file" <<EOF
-- Migration for: $title (UP)

-- Example: CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100));

EOF

# Repeat for the down migration file
cat >"$down_file" <<EOF
-- Migration for: $title (DOWN)

-- Example: DROP TABLE users;

EOF

echo "Migration files created:"
echo "$up_file"
echo "$down_file"
