go build -o abyssnet.dll -buildmode=c-shared .
Copy-Item abyssnet.dll ..\abyss_engine\bin\Debug\net8.0\
#Copy-Item abyssnet.dll ..\abyss_engine\bin\Release\net8.0\