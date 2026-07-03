import argparse
import os
from dataclasses import dataclass

import yaml


@dataclass(frozen=True)
class Config:
    grpc_addr: str
    health_addr: str
    backend: str
    remote_endpoint: str
    remote_model: str
    api_key: str


def _deep_merge(base: dict, overlay: dict) -> dict:
    result = dict(base)
    for key, value in overlay.items():
        if isinstance(value, dict) and isinstance(result.get(key), dict):
            result[key] = _deep_merge(result[key], value)
        else:
            result[key] = value
    return result


def load() -> Config:
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", default="configs")
    parser.add_argument("--env", default="dev")
    args, _ = parser.parse_known_args()

    merged: dict = {}
    for path in (
        os.path.join(args.config, "base", "common.yml"),
        os.path.join(args.config, "base", "embed.yml"),
        os.path.join(args.config, args.env, "embed.yml"),
        os.path.join(args.config, "secrets.yml"),
    ):
        if os.path.exists(path):
            with open(path, encoding="utf-8") as handle:
                merged = _deep_merge(merged, yaml.safe_load(handle) or {})

    embed = merged.get("embed", {})
    remote = embed.get("remote", {})
    return Config(
        grpc_addr=merged.get("grpc", {}).get("addr", ":9097"),
        health_addr=merged.get("health", {}).get("addr", ":8097"),
        backend=embed.get("backend", "deterministic"),
        remote_endpoint=remote.get("base_url", "") or remote.get("endpoint", ""),
        remote_model=remote.get("model", "bge-m3"),
        api_key=remote.get("api_key", ""),
    )
