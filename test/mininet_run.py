#!/usr/bin/env python2
""" This script runs the tests for this project.
Run it in a mininet environment with the following command:
sudo ./mininet_run.py

or

sudo ./mininet_run -d

for debug-level output"""

from functools import partial
from mininet.topo import SingleSwitchTopo
from mininet.net import Mininet
from mininet.node import Host, OVSController
from optparse import OptionParser
from mininet.log import setLogLevel
from mininet.cli import CLI


def run_tests(cli):
    "Create and test a simple network"
    topo = SingleSwitchTopo(k=3)
    private_dirs = ['/run', ('/var/run', '/tmp/%(name)s/var/run'), '/var/mn']
    host = partial(Host, privateDirs=private_dirs)
    net = Mininet(topo=topo, host=host, controller=OVSController)
    net.start()
    if cli:
        CLI(net)
    else:
        h1 = net.get("h1")
        print h1.cmd("go test")
    net.stop()


if __name__ == '__main__':
    parser = OptionParser()
    parser.add_option("-d", "--debug", dest="debug",
                      action="store_true", default=False)
    parser.add_option("-c", "--cli", dest="cli",
                      action="store_true", default=False)
    options, args = parser.parse_args()
    if options.debug:
        setLogLevel("info")

    run_tests(options.cli)
