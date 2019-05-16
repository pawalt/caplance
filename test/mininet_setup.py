#!/usr/bin/env python2
""" This script provides a basic switch topo."""

from functools import partial
from mininet.topo import SingleSwitchTopo
from mininet.net import Mininet
from mininet.cli import CLI
from mininet.node import Host, OVSController
from mininet.log import setLogLevel


def simple_test():
    "Create and test a simple network"
    topo = SingleSwitchTopo(k=2)
    private_dirs = ['/run', ('/var/run', '/tmp/%(name)s/var/run'), '/var/mn']
    host = partial(Host, privateDirs=private_dirs)
    net = Mininet(topo=topo, host=host, controller=OVSController)
    net.start()
    CLI(net)
    net.stop()


if __name__ == '__main__':
    # Tell mininet to print useful information
    setLogLevel('info')
    simple_test()
