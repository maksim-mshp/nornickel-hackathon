import argparse
import os
from dataclasses import dataclass

import yaml


@dataclass(frozen=True)
class S3Config:
    endpoint: str
    access_key: str
    secret_key: str
    use_ssl: bool
    bundles_bucket: str


@dataclass(frozen=True)
class Config:
    nats_url: str
    s3: S3Config


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
        os.path.join(args.config, args.env, "common.yml"),
        os.path.join(args.config, "secrets.yml"),
    ):
        if os.path.exists(path):
            with open(path, encoding="utf-8") as handle:
                merged = _deep_merge(merged, yaml.safe_load(handle) or {})

    s3 = merged.get("s3", {})
    buckets = s3.get("buckets", {})
    return Config(
        nats_url=merged.get("nats", {}).get("url", "nats://localhost:4222"),
        s3=S3Config(
            endpoint=s3.get("endpoint", "localhost:9000"),
            access_key=s3.get("access_key", "kmap"),
            secret_key=s3.get("secret_key", "kmap-minio-secret"),
            use_ssl=bool(s3.get("use_ssl", False)),
            bundles_bucket=buckets.get("bundles", "kmap-bundles"),
        ),
    )
