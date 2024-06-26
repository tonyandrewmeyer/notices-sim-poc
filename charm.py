#! /usr/bin/env python3

"""Pretend to be a charm."""

import logging

import click
import ops.pebble

logger = logging.getLogger(__name__)


@click.command()
@click.argument("event_type", type=str)
@click.argument("notice_id", type=str)
@click.argument("notice_type", type=str)
@click.argument("notice_key", type=str)
def main(event_type, notice_id, notice_type, notice_key):
    """Pretend to be a charm."""
    handler = logging.FileHandler("./charm.log")
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)
    client = ops.pebble.Client("/tmp/pebble/.pebble.socket")
    if event_type == "custom":
        assert notice_type == ops.pebble.NoticeType.CUSTOM.value
        notice = client.get_notice(notice_id)
        logger.info(
            "Custom: id=%s, user_id=%s, key=%s, occurrences=%s, last_data=%s",
            notice.id,
            notice.user_id,
            notice.key,
            notice.occurrences,
            notice.last_data,
        )
    elif event_type == "change-updated":
        change = client.get_change(notice_key)
        logger.info(
            "A change changed: id=%s, kind=%s, summary=%s, status=%s, ready=%s, err=%s, data=%s",
            change.id,
            change.kind,
            change.summary,
            change.status,
            change.ready,
            change.err,
            change.data,
        )
    elif event_type == "recover-check":
        assert notice_type == "change-update"
        for check in client.get_checks():
            if check.change_id == notice_key:
                logger.info("Check %r is %s (failure count: %s/%s)", check.name, check.status, check.failures, check.threshold)
    elif event_type == "perform-check":
        assert notice_type == "change-update"
        for check in client.get_checks():
            if check.change_id == notice_key:
                logger.info("Check %r is %s (failure count: %s/%s)", check.name, check.status, check.failures, check.threshold)

    # Uncomment to make things badly loop.
#    client.replan_services()


if __name__ == "__main__":
    main()
