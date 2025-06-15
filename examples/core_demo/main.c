#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>

int main(int argc, char *argv[])
{
    while (1)
    {
        printf("Current process PID: %d\n", getpid());
        sleep(1);
    }
    return 0;
}
