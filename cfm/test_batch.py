#!/usr/bin/env python3
"""
批量预测测试脚本
用于测试 /predict/batch 端点
"""

import requests
import sys
import os

def test_batch_predict(input_file="example_input.txt", output_file="test_results.xlsx", prob_thresh="0.001"):
    """
    测试批量预测功能
    
    Args:
        input_file: 输入文件路径
        output_file: 输出 Excel 文件路径
        prob_thresh: 概率阈值
    """
    url = "http://localhost:5001/predict/batch"
    params = {"prob_thresh": prob_thresh}
    
    if not os.path.exists(input_file):
        print(f"错误: 输入文件 '{input_file}' 不存在")
        return False
    
    print(f"📤 上传文件: {input_file}")
    print(f"📊 概率阈值: {prob_thresh}")
    print(f"⏳ 正在处理...")
    
    try:
        with open(input_file, "rb") as f:
            files = {"file": (os.path.basename(input_file), f, "text/plain")}
            response = requests.post(url, params=params, files=files, timeout=300)
        
        if response.status_code == 200:
            with open(output_file, "wb") as out:
                out.write(response.content)
            file_size = os.path.getsize(output_file)
            print(f"✅ 预测完成！")
            print(f"📁 结果已保存到: {output_file}")
            print(f"📏 文件大小: {file_size / 1024:.2f} KB")
            return True
        else:
            print(f"❌ 错误: HTTP {response.status_code}")
            print(f"响应内容: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ 错误: 无法连接到服务器")
        print("   请确保服务正在运行: http://localhost:5001")
        return False
    except requests.exceptions.Timeout:
        print("❌ 错误: 请求超时（可能分子数量太多）")
        return False
    except Exception as e:
        print(f"❌ 错误: {e}")
        return False

if __name__ == "__main__":
    # 解析命令行参数
    input_file = sys.argv[1] if len(sys.argv) > 1 else "example_input.txt"
    output_file = sys.argv[2] if len(sys.argv) > 2 else "test_results.xlsx"
    prob_thresh = sys.argv[3] if len(sys.argv) > 3 else "0.001"
    
    print("=" * 50)
    print("CFM-ID 批量预测测试")
    print("=" * 50)
    
    success = test_batch_predict(input_file, output_file, prob_thresh)
    
    sys.exit(0 if success else 1)

