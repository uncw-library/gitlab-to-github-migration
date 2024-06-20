from datetime import datetime
import logging
import os

import constants


def setup_logging():
    logging_dir = os.path.join(constants.APP_ROOT, "logs")
    os.makedirs(logging_dir, exist_ok=True)
    logging.basicConfig(
        filename=os.path.join(logging_dir, f"output-{datetime.now().strftime('%Y%m%d-%H-%M-%S')}.log"),
        filemode="w",
        format="%(name)s - %(levelname)s - %(message)s",
        level=logging.INFO,
    )
