FROM python:3.13-slim
RUN useradd -u 10001 -m kmap
WORKDIR /app
COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .
USER kmap
ENTRYPOINT ["kmap-embed"]
