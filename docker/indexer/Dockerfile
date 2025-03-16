FROM dipdup/dipdup:8
# FROM ghcr.io/dipdup-io/dipdup:8
# FROM ghcr.io/dipdup-io/dipdup:next

# COPY --chown=dipdup pyproject.toml README.md .
# RUN pip install .

COPY --chown=dipdup indexer indexer_workspace

WORKDIR indexer_workspace