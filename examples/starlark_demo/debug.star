# 定义一个函数来打印当前作用域的信息
def print_scope():
    scope = cur_scope()
    print("Current scope:", scope)
    dlv_command("locals")

# 定义一个函数来设置断点并执行调试命令
def debug_person():
    # 打印当前作用域
    print_scope()
    
    # 打印变量 p 的值
    dlv_command("print p")
    
    # 单步执行
    dlv_command("next")
    
    # 再次打印作用域
    print_scope()

# 定义一个函数来保存调试信息到文件
def save_debug_info():
    # 获取当前作用域
    scope = cur_scope()
    
    # 将调试信息写入文件
    debug_info = "Debug session at " + str(time.time()) + "\n"
    debug_info += "Current scope: " + str(scope) + "\n"
    
    # 保存到文件
    write_file("debug_info.txt", debug_info)

# 主函数
def main():
    print("Starting debug session...")
    
    # 设置断点
    dlv_command("break main.main")
    dlv_command("break main.processPerson")
    
    # 继续执行到main.main
    dlv_command("continue")
    
    # 继续执行到main.processPerson
    dlv_command("continue")
 
    # 执行调试操作
    debug_person()
    
    # 保存调试信息
    save_debug_info()
    
    print("Debug session completed.")

# 直接调用 main 函数 (source命令会自动调用定义的 `main` 函数)
#main() 