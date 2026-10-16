local total = 0
for i = 1, 3000000 do total = total + i % 7 end
print(total)
