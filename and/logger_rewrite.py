import re

def replace_w_stat(input_file):
    # Read the file content
    with open(input_file, 'r') as f:
        content = f.read()

    # Replace all occurrences of w.o.stat.W(N) with w.o.stat.W(0)
    updated_content = re.sub(r'w\.o\.stat\.W\(\d+\)', 'w.o.stat.W(0)', content)

    with open(input_file, 'w') as f:
        f.write(updated_content)

def enumerate_w_stat(input_file):
    # Read the file content
    with open(input_file, 'r') as f:
        content = f.read()

    # Function to incrementally replace each match
    counter = 0
    def replace_func(match):
        nonlocal counter
        replacement = f"w.o.stat.W({counter})"
        counter += 1
        return replacement

    # Replace each 'w.o.stat.W(0)' with 'w.o.stat.W(n)' where n increments
    updated_content = re.sub(r'w\.o\.stat\.W\(0\)', replace_func, content)

    with open(input_file, 'w') as f:
        f.write(updated_content)

# Example usage:
replace_w_stat('world.go')  # Replace 'input.txt' with your filename
enumerate_w_stat('world.go')