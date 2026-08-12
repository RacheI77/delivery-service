#!/usr/bin/env bash
set -e

# Auto-create .env from template if missing
if [ ! -f .env ]; then
    echo "No .env found. Creating one from .env.example ..."
    cp .env.example .env
fi

# If the Google Maps API key is still the placeholder, ask the user for a real one
if grep -q 'YOUR_ACTUAL_API_KEY' .env; then
    echo ""
    echo "The GOOGLE_MAPS_API_KEY in .env is still the placeholder."
    echo "You need a real key (with the Distance Matrix API enabled) to use the order placement (distance) feature."
    read -r -p "Enter your Google Maps API key (or press Enter to skip and use the placeholder): " api_key
    if [ -n "$api_key" ]; then
        # Replace the placeholder value in .env with the user-provided key
        sed -i "s|^GOOGLE_MAPS_API_KEY=.*|GOOGLE_MAPS_API_KEY=${api_key}|" .env
        echo "GOOGLE_MAPS_API_KEY updated in .env."
    else
        echo "Skipped. The placeholder key will be used; order placement (distance) will fail until you set a real key in .env."
    fi
fi

echo ""
echo "Starting application environment..."
docker-compose down -v
docker-compose up --build -d

echo "Waiting for service to be ready on port 8080..."
until [ "$(curl -s -o /dev/null -w '%{http_code}' 'http://localhost:8080/orders?page=1&limit=1')" = "200" ]; do
    printf '.'
    sleep 1
done

echo -e "\nApplication is live and accessible at http://localhost:8080"