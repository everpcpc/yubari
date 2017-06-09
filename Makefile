init:
	pyvenv-3.5 venv
	venv/bin/pip install -r pip-req.txt

install_venv:
	venv/bin/python setup.py install

install:
	python setup.py install
