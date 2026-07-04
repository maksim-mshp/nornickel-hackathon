FROM python:3.13-slim
RUN useradd -u 10001 -m kmap
WORKDIR /app
ENV HF_HOME=/opt/hf
RUN pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu
COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .
RUN python -c "from transformers import AutoModel, AutoTokenizer; AutoTokenizer.from_pretrained('BAAI/bge-m3'); AutoModel.from_pretrained('BAAI/bge-m3')" \
    && chown -R kmap:kmap /opt/hf
ENV HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1 OMP_NUM_THREADS=4
USER kmap
ENTRYPOINT ["kmap-embed"]
