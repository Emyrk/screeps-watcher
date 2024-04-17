package workdemo

var workAmount = 100000000

func CallStackOne(count int) {
	for i := 0; i < workAmount; i++ {
		count += i
		if i%2 == 0 {
			count = count / 2
		}
	}
	CallStackTwo(count)
}

func CallStackTwo(count int) {
	for i := 0; i < workAmount; i++ {
		count += i
		if i%2 == 0 {
			count = count / 2
		}
	}
	CallStackThree(count)
}

func CallStackThree(count int) {
	for i := 0; i < workAmount; i++ {
		count += i
		if i%2 == 0 {
			count = count / 2
		}
	}
	CallStackFour(count)
}

func CallStackFour(count int) {
	for i := 0; i < workAmount; i++ {
		count += i
		if i%2 == 0 {
			count = count / 2
		}
	}
}
