import ipaddress
import socket
from collections.abc import Callable
from urllib.parse import urlparse


Resolver = Callable[[str], list[str]]


def system_resolver(hostname: str) -> list[str]:
    return sorted({item[4][0] for item in socket.getaddrinfo(hostname, None)})


def validate_public_url(raw_url: str, resolver: Resolver = system_resolver) -> str:
    parsed = urlparse(raw_url)
    if parsed.scheme not in {"http", "https"}:
        raise ValueError("only HTTP and HTTPS URLs are allowed")
    if not parsed.hostname:
        raise ValueError("URL hostname is required")
    addresses = resolver(parsed.hostname)
    if not addresses:
        raise ValueError("hostname has no resolved addresses")
    for address in addresses:
        ip = ipaddress.ip_address(address)
        if not ip.is_global:
            raise ValueError("target resolves to a non-public address")
    return raw_url


def validate_redirect_chain(urls: list[str], resolver: Resolver = system_resolver) -> None:
    if len(urls) > 10:
        raise ValueError("redirect chain exceeds limit")
    for url in urls:
        validate_public_url(url, resolver)

