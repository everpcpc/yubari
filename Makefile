init:
	pyvenv-3.5 venv
	venv/bin/pip install -r pip-req.txt

install:
	venv/bin/python setup.py install
