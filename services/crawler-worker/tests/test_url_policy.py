import pytest
from opportunity_crawler.url_policy import validate_public_url, validate_redirect_chain


def test_blocks_private_and_metadata_targets() -> None:
    for address in ["127.0.0.1", "10.0.0.1", "169.254.169.254", "::1"]:
        with pytest.raises(ValueError):
            validate_public_url("https://untrusted.test/path", lambda _: [address])


def test_allows_public_http_target_and_rechecks_redirects() -> None:
    assert validate_public_url("https://example.test", lambda _: ["93.184.216.34"])
    with pytest.raises(ValueError):
        validate_redirect_chain(
            ["https://example.test", "http://internal.test"],
            lambda host: ["93.184.216.34"] if host == "example.test" else ["192.168.1.10"],
        )
