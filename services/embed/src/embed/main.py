import logging
import threading
from concurrent import futures
from http.server import BaseHTTPRequestHandler, HTTPServer

import grpc
from grpc_reflection.v1alpha import reflection

from embed.config import Config, load
from embed.embedder import (
    CachingEmbedder,
    DeterministicEmbedder,
    LocalReranker,
    RemoteEmbedder,
    RemoteReranker,
)
from embed.server import EmbedServicer
from kmap.v1 import embed_pb2, embed_pb2_grpc

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("embed")


def _bind(addr: str) -> str:
    return "0.0.0.0" + addr if addr.startswith(":") else addr


def _build_backend(cfg: Config):
    if cfg.backend == "remote" and cfg.api_key and cfg.remote_endpoint:
        logger.info("embed backend: remote (%s)", cfg.remote_model)
        inner = RemoteEmbedder(cfg.remote_endpoint, cfg.api_key, cfg.remote_model)
    else:
        if cfg.backend == "remote":
            logger.warning("embed backend: remote configured but no key/endpoint — offline fallback to deterministic")
        else:
            logger.info("embed backend: deterministic")
        inner = DeterministicEmbedder()
    return CachingEmbedder(inner, cfg.cache_size)


def _build_reranker(cfg: Config):
    if cfg.backend == "remote" and cfg.api_key and cfg.remote_endpoint:
        logger.info("rerank backend: remote (%s)", cfg.reranker_model)
        return RemoteReranker(cfg.remote_endpoint, cfg.api_key, cfg.reranker_model)
    logger.info("rerank backend: local token-overlap")
    return LocalReranker()


def _start_health(addr: str) -> None:
    host, _, port = _bind(addr).partition(":")

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"service":"embed","status":"ok"}\n')

        def log_message(self, *args):
            return

    server = HTTPServer((host, int(port)), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()


def main() -> None:
    cfg = load()
    backend = _build_backend(cfg)
    reranker = _build_reranker(cfg)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    embed_pb2_grpc.add_EmbedServiceServicer_to_server(EmbedServicer(backend, reranker), server)
    reflection.enable_server_reflection(
        (embed_pb2.DESCRIPTOR.services_by_name["EmbedService"].full_name, reflection.SERVICE_NAME),
        server,
    )
    server.add_insecure_port(_bind(cfg.grpc_addr))
    server.start()
    _start_health(cfg.health_addr)
    logger.info("embed grpc listening on %s", cfg.grpc_addr)
    server.wait_for_termination()


if __name__ == "__main__":
    main()
