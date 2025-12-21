# Agent Development Guidelines

## Python Execution

**IMPORTANT**: Always use `uv` for Python operations in this project. Never invoke Python directly.

### Running Python Scripts
```bash
# Correct
uv run script.py
uv run python -m module_name

# Incorrect - DO NOT USE
python script.py
python3 script.py
./script.py
```

### Running Tests
```bash
# Correct
uv run pytest
uv run pytest test_file.py
uv run pytest -v

# Incorrect - DO NOT USE
pytest
python -m pytest
```

### Installing Dependencies
```bash
# Correct
uv add package_name
uv sync

# Incorrect - DO NOT USE
pip install package_name
python -m pip install
```

## Project Structure
This project uses `uv` as the Python package manager and virtual environment tool. The `pyproject.toml` file contains all dependencies and project configuration.