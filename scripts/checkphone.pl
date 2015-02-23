#!/usr/bin/perl -w

use Socket;
use strict;
my $fh;

my $zmqclient = "/flash/tuer/door_client_zmq";

my $keys;
my %good;

open $keys,'/flash/phone';
while (<$keys>)
{
	chomp;
	if ($_ =~ /^(\S+)\s+(.+)$/)
	{
		$good{$1}=$2;
 	}
}
my $id = $ARGV[1];
$id =~ s/^0/+43/;
my $action = $ARGV[0];
if ($good{$id})
{
	if ($action == 1591)
        {
	  send_to_fifo("close Phone ".$good{$id});
        } elsif ($action == 1590) {
	  send_to_fifo("open Phone ".$good{$id});
        } elsif ($action == 1592) {
	  send_to_fifo("reset Phone ".$good{$id});
        } else {
          send_to_fifo("log invalid action $action, but valid Phone $id");
        }
} else {
	send_to_fifo("log invalid Phone $id");
}
exit 0;

sub send_to_fifo
{
  open(my $conn, "| ".$zmqclient);
	#socket(my $conn, PF_UNIX, SOCK_STREAM,0) || die "socket: $!";
	#connect($conn, $socketaddr) || die "socket connect: $!";
	print $conn shift(@_)."\n";
	close($conn);
}

