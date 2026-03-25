// staticlint — мультичекер, объединяющий несколько инструментов статического анализа
// в один бинарник. Запускается со стандартными флагами go/analysis:
//
//	staticlint ./...
//
// # Включённые анализаторы
//
// ## Стандартные проходы (golang.org/x/tools/go/analysis/passes)
//
// Включены следующие анализаторы из стандартного набора инструментов Go:
//
//   - appends      — обнаруживает отсутствие значений после append
//   - assign       — обнаруживает бесполезные присваивания
//   - atomic       — проверяет типичные ошибки при использовании sync/atomic
//   - bools        — обнаруживает типичные ошибки с булевыми операторами
//   - buildtag     — проверяет корректность build-тегов
//   - cgocall      — обнаруживает нарушения правил передачи указателей CGo
//   - composite    — помечает составные литералы с неименованными полями
//   - copylock     — сообщает о блокировках, переданных или присвоенных по значению
//   - defers       — сообщает о типичных ошибках в операторах defer
//   - directive    — проверяет директивы инструментария Go, например //go:generate
//   - errorsas     — проверяет, что второй аргумент errors.As реализует error
//   - httpresponse — обнаруживает ошибки при работе с HTTP-ответами
//   - ifaceassert  — помечает невозможные утверждения интерфейс-к-интерфейсу
//   - loopclosure  — проверяет захват переменной цикла во вложенных функциях
//   - lostcancel   — обнаруживает функции cancel из context.WithCancel, которые никогда не вызываются
//   - nilfunc      — помечает сравнения функций с nil
//   - nilness      — сообщает об избыточных проверках на nil
//   - printf       — проверяет согласованность форматных строк семейства Printf
//   - shadow       — проверяет затенение переменных
//   - shift        — помечает сдвиги, превышающие ширину целого числа
//   - sigchanyzer  — проверяет неправильное использование os/signal.Notify
//   - slog         — проверяет некорректные вызовы структурированного логирования
//   - sortslice    — проверяет вызовы sort.Slice с корректной арностью функции less
//   - stdmethods   — проверяет корректность сигнатур методов стандартных интерфейсов
//   - stringintconv — помечает преобразования string(int)
//   - structtag    — проверяет корректность тегов полей структур
//   - testinggoroutine — сообщает о вызовах t.Fatal из горутин, запущенных тестом
//   - tests        — проверяет типичные ошибки в именовании Test/Example/Benchmark
//   - timeformat   — проверяет форматные строки времени
//   - unmarshal    — проверяет некорректные вызовы json/xml Unmarshal
//   - unreachable  — обнаруживает недостижимый код
//   - unsafeptr    — проверяет преобразования unsafe.Pointer
//   - unusedresult — помечает неиспользуемые возвращаемые значения выбранных чистых функций
//   - unusedwrite  — помечает бесполезные записи только для записи
//   - waitgroup    — проверяет использование sync.WaitGroup Add/Done
//
// ## Класс SA из staticcheck (honnef.co/go/tools/staticcheck)
//
// Включены все SA*-анализаторы. Они обнаруживают реальные баги и
// некорректное использование API, например:
//
//   - SA1000  некорректные регулярные выражения
//   - SA1001  некорректные форматные строки
//   - SA4006  неиспользуемые переменные
//   - SA5000  некорректные преобразования unsafe.Pointer
//   - …и многие другие (см. https://staticcheck.dev/docs/checks/#SA)
//
// ## Класс S из staticcheck (honnef.co/go/tools/simple)
//
// Все S*-анализаторы предлагают упрощения кода, например:
//
//   - S1000  заменить select/channel на обычный приём из канала
//   - S1001  заменить цикл встроенным copy
//   - S1023  убрать избыточный оператор return
//   - …и другие (см. https://staticcheck.dev/docs/checks/#S)
//
// ## Публичные анализаторы
//
//   - ineffassign (github.com/gordonklaus/ineffassign)
//     Обнаруживает присваивания переменным, которые никогда не используются после присваивания.
//
//   - bodyclose (github.com/timakin/bodyclose)
//     Проверяет, что тела HTTP-ответов корректно закрываются,
//     предотвращая утечки ресурсов в HTTP-клиентском коде.
//
// ## Собственный анализатор
//
//   - osexit (github.com/glebb1331/shortener-practicum/cmd/staticlint/osexit)
//     Запрещает прямые вызовы os.Exit внутри функции main пакета main.
//     Прямые вызовы os.Exit обходят отложенную очистку и затрудняют тестирование.
//     Используйте log.Fatal или возвращайте ошибку из вспомогательной функции run().
//
// # Запуск
//
//	# анализ всего модуля
//	go run ./cmd/staticlint/... ./...
//
//	# сначала собрать, затем запустить
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
//	# показать доступные флаги
//	./staticlint -help
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	ineffassign "github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/timakin/bodyclose/passes/bodyclose"

	lintanalyzers "honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"

	"github.com/glebb1331/shortener-practicum/cmd/staticlint/osexit"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// Стандартные проходы.
		appends.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		waitgroup.Analyzer,

		// Публичные анализаторы.
		ineffassign.Analyzer,
		bodyclose.Analyzer,

		// Собственный анализатор.
		osexit.Analyzer,
	}

	// Добавляем все SA-анализаторы staticcheck.
	analyzers = append(analyzers, unwrap(staticcheck.Analyzers)...)
	// Добавляем все S*-анализаторы (класс simple) — удовлетворяет требованию «не менее одного не-SA класса».
	analyzers = append(analyzers, unwrap(simple.Analyzers)...)

	multichecker.Main(analyzers...)
}

// unwrap преобразует срез обёрток lint.Analyzer из staticcheck в указатели
// на golang.org/x/tools/go/analysis.Analyzer, ожидаемые multichecker.Main.
func unwrap(as []*lintanalyzers.Analyzer) []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, len(as))
	for i, a := range as {
		out[i] = a.Analyzer
	}
	return out
}
