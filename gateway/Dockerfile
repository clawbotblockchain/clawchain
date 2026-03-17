FROM python:3.12-slim

WORKDIR /app

# Install clawchaind binary (copy from builder or mount)
# The binary must be available in PATH or set CLAWCHAIND_PATH env var
COPY --from=golang:1.24 /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy gateway source
COPY . .

# Create data directory for SQLite
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["uvicorn", "gateway.main:app", "--host", "0.0.0.0", "--port", "8080"]
