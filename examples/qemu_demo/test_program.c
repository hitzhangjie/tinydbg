#include <stdio.h>
#include <unistd.h>

int global_var = 42;

int add(int a, int b) {
    return a + b;
}

int main() {
    int local_var = 10;
    printf("Starting test program...\n");
    
    for (int i = 0; i < 5; i++) {
        printf("Loop iteration %d\n", i);
        int result = add(i, global_var);
        printf("Result: %d\n", result);
        sleep(1);  // 添加延时，方便观察
    }
    
    printf("Program finished.\n");
    return 0;
} 