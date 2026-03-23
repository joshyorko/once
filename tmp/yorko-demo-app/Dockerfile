FROM python:3.12-alpine

WORKDIR /app

COPY server.py /app/server.py
COPY index.html /app/index.html

EXPOSE 80

CMD ["python", "/app/server.py"]
