package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func predictHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad form data", http.StatusBadRequest)
		return
	}

	// Read parameters
	prob := r.FormValue("prob_thresh")
	if prob == "" {
		prob = "0.001"
	}

	smiles := r.FormValue("smiles")
	if smiles == "" {
		http.Error(w, "Missing 'smiles' parameter", http.StatusBadRequest)
		return
	}

	// Create input file in file-mode: need an ID and SMILES per line
	inFile, err := os.CreateTemp("", "cfm-in-*.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("Create input file failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(inFile.Name())
	// Write ID (M1) and SMILES
	inFile.WriteString(fmt.Sprintf("M1 %s\n", smiles))
	inFile.Close()

	// Create temp output file
	outFile, err := os.CreateTemp("", "cfm-out-*.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("Create output file failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(outFile.Name())
	outFile.Close()

	// Run CFM-ID in file mode
	cmd := exec.Command(
		"cfm-predict",
		inFile.Name(), // input file with ID SMILES
		prob,          // prob_thresh
		"/trained_models_cfmid4.0/cfmid4/[M+H]+/param_output.log",
		"/trained_models_cfmid4.0/cfmid4/[M+H]+/param_config.txt",
		"1",            // annotate_fragments = YES
		outFile.Name(), // output file
		"1",            // apply_postproc
		"0",            // suppress_exceptions
	)

	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("cfm-predict failed: %v", err), http.StatusInternalServerError)
		return
	}

	result, err := os.ReadFile(outFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Read result failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(result)
}

// PredictionResult 存储单个分子的预测结果
type PredictionResult struct {
	ID        string
	SMILES    string
	InChiKey  string
	Formula   string
	PMass     string
	Fragments []Fragment
}

// Fragment 存储碎片信息
type Fragment struct {
	EnergyLevel int
	MZ          float64
	Intensity   float64
	FragmentID  int
	Annotation  string
}

// parseCFMOutput 解析 CFM-ID 的输出文件
func parseCFMOutput(outputContent string) ([]PredictionResult, error) {
	var results []PredictionResult
	var currentResult *PredictionResult
	var currentEnergyLevel int = -1

	scanner := bufio.NewScanner(strings.NewReader(outputContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			// 解析元数据
			if strings.HasPrefix(line, "#ID=") {
				// 如果遇到新的 ID，先保存前一个结果
				if currentResult != nil && currentResult.ID != "" {
					results = append(results, *currentResult)
				}
				id := strings.TrimPrefix(line, "#ID=")
				currentResult = &PredictionResult{ID: id, Fragments: []Fragment{}}
				currentEnergyLevel = -1 // 重置能量级别
			} else if strings.HasPrefix(line, "#SMILES=") {
				if currentResult != nil {
					currentResult.SMILES = strings.TrimPrefix(line, "#SMILES=")
				}
			} else if strings.HasPrefix(line, "#InChiKey=") {
				if currentResult != nil {
					currentResult.InChiKey = strings.TrimPrefix(line, "#InChiKey=")
				}
			} else if strings.HasPrefix(line, "#Formula=") {
				if currentResult != nil {
					currentResult.Formula = strings.TrimPrefix(line, "#Formula=")
				}
			} else if strings.HasPrefix(line, "#PMass=") {
				if currentResult != nil {
					currentResult.PMass = strings.TrimPrefix(line, "#PMass=")
				}
			}
			continue
		}

		// 检测能量级别
		if strings.HasPrefix(line, "energy") {
			levelStr := strings.TrimPrefix(line, "energy")
			if level, err := strconv.Atoi(levelStr); err == nil {
				currentEnergyLevel = level
			}
			continue
		}

		// 解析碎片数据行（格式：m/z intensity fragment_id (annotation)）
		// 例如：55.05423 11.21 19 (9.1697)
		parts := strings.Fields(line)
		if len(parts) >= 3 && currentResult != nil && currentEnergyLevel >= 0 {
			mz, err1 := strconv.ParseFloat(parts[0], 64)
			intensity, err2 := strconv.ParseFloat(parts[1], 64)
			fragmentID, err3 := strconv.Atoi(parts[2])
			if err1 == nil && err2 == nil && err3 == nil {
				annotation := ""
				if len(parts) > 3 {
					// 提取括号中的注释
					re := regexp.MustCompile(`\(([^)]+)\)`)
					matches := re.FindStringSubmatch(line)
					if len(matches) > 1 {
						annotation = matches[1]
					}
				}
				currentResult.Fragments = append(currentResult.Fragments, Fragment{
					EnergyLevel: currentEnergyLevel,
					MZ:          mz,
					Intensity:   intensity,
					FragmentID:  fragmentID,
					Annotation:  annotation,
				})
			}
		}
	}

	// 保存最后一个结果
	if currentResult != nil && currentResult.ID != "" {
		results = append(results, *currentResult)
	}

	return results, scanner.Err()
}

// exportToExcel 将预测结果导出为 Excel
func exportToExcel(results []PredictionResult, filename string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Predictions"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"ID", "SMILES", "InChiKey", "Formula", "PMass", "Energy Level", "m/z", "Intensity", "Fragment ID", "Annotation"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)

	// 填充数据
	row := 2
	for _, result := range results {
		if len(result.Fragments) == 0 {
			// 如果没有碎片，至少输出基本信息
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), result.ID)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), result.SMILES)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), result.InChiKey)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), result.Formula)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), result.PMass)
			row++
		} else {
			for _, frag := range result.Fragments {
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), result.ID)
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), result.SMILES)
				f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), result.InChiKey)
				f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), result.Formula)
				f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), result.PMass)
				f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), frag.EnergyLevel)
				f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), frag.MZ)
				f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), frag.Intensity)
				f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), frag.FragmentID)
				f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), frag.Annotation)
				row++
			}
		}
	}

	// 自动调整列宽
	for i := 0; i < len(headers); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// 删除默认的 Sheet1
	f.DeleteSheet("Sheet1")

	return f.SaveAs(filename)
}

// batchPredictHandler 处理批量预测请求
func batchPredictHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取 prob_thresh 参数
	prob := r.URL.Query().Get("prob_thresh")
	if prob == "" {
		prob = "0.001"
	}

	// 解析上传的文件
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, fmt.Sprintf("Parse form failed: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Read file failed: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Read file content failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 解析输入文件（支持两种格式：每行一个SMILES，或 ID SMILES）
	var molecules []struct {
		ID     string
		SMILES string
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// 格式：ID SMILES
			molecules = append(molecules, struct {
				ID     string
				SMILES string
			}{ID: parts[0], SMILES: strings.Join(parts[1:], " ")})
		} else if len(parts) == 1 {
			// 格式：只有 SMILES，自动生成 ID
			molecules = append(molecules, struct {
				ID     string
				SMILES string
			}{ID: fmt.Sprintf("M%d", lineNum), SMILES: parts[0]})
		}
		lineNum++
	}

	if len(molecules) == 0 {
		http.Error(w, "No valid molecules found in file", http.StatusBadRequest)
		return
	}

	// 创建输入文件
	inFile, err := os.CreateTemp("", "cfm-batch-in-*.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("Create input file failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(inFile.Name())

	for _, mol := range molecules {
		inFile.WriteString(fmt.Sprintf("%s %s\n", mol.ID, mol.SMILES))
	}
	inFile.Close()

	// 创建输出文件
	outFile, err := os.CreateTemp("", "cfm-batch-out-*.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("Create output file failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(outFile.Name())
	outFile.Close()

	// 运行 CFM-ID
	cmd := exec.Command(
		"cfm-predict",
		inFile.Name(),
		prob,
		"/trained_models_cfmid4.0/cfmid4/[M+H]+/param_output.log",
		"/trained_models_cfmid4.0/cfmid4/[M+H]+/param_config.txt",
		"1",
		outFile.Name(),
		"1",
		"0",
	)

	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("cfm-predict failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 读取结果
	resultContent, err := os.ReadFile(outFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Read result failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 解析结果
	results, err := parseCFMOutput(string(resultContent))
	if err != nil {
		http.Error(w, fmt.Sprintf("Parse result failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 生成 Excel 文件
	excelFile, err := os.CreateTemp("", "cfm-results-*.xlsx")
	if err != nil {
		http.Error(w, fmt.Sprintf("Create Excel file failed: %v", err), http.StatusInternalServerError)
		return
	}
	excelFile.Close()
	defer os.Remove(excelFile.Name())

	if err := exportToExcel(results, excelFile.Name()); err != nil {
		http.Error(w, fmt.Sprintf("Export to Excel failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 读取 Excel 文件内容
	excelContent, err := os.ReadFile(excelFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Read Excel file failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 设置响应头
	filename := "cfm_predictions.xlsx"
	if header.Filename != "" {
		baseName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
		filename = baseName + "_results.xlsx"
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(excelContent)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/predict", predictHandler)
	mux.HandleFunc("/predict/batch", batchPredictHandler)

	srv := &http.Server{
		Addr:    ":5001",
		Handler: mux,
	}

	log.Println("🚀 CFM-ID wrapper starting on http://0.0.0.0:5001")
	log.Fatal(srv.ListenAndServe())
}
