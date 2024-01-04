from flask import Flask, request, jsonify
from transformers import AutoModelForCausalLM, AutoTokenizer
import json

app = Flask(__name__)

model_name_or_path = "TheBloke/Llama-2-13B-chat-GPTQ"
model = AutoModelForCausalLM.from_pretrained(model_name_or_path,
                                             device_map="auto",
                                             trust_remote_code=False,
                                             revision="main")
tokenizer = AutoTokenizer.from_pretrained(model_name_or_path, use_fast=True)

def clean_up_response(response):
    # Find the delimiter that indicates the start of the JSON content
    json_start = response.find("<</SYS>>[/INST]") + len("<</SYS>>[/INST]")
    
    # If the delimiter was found, proceed to extract the JSON
    if json_start > len("<</SYS>>[/INST]") - 1:
        # The JSON starts after the delimiter; extract it from there
        json_str = response[json_start:]
        
        # Find the first occurrence of the opening '{' which marks the beginning of the JSON object
        start_of_json = json_str.find("{")
        # Find the last occurrence of the closing '}' which marks the end of the JSON object
        end_of_json = json_str.rfind("}")

        if start_of_json != -1 and end_of_json != -1:
            # Extract the JSON string
            json_str = json_str[start_of_json:end_of_json+1]

            try:
                # Parse the JSON string into a Python dictionary
                json_obj = json.loads(json_str)

                # Pretty-print the JSON object with an indent of 4 spaces
                formatted_json_str = json.dumps(json_obj, indent=4)

                # Return the cleaned, formatted JSON string
                return formatted_json_str
            except json.JSONDecodeError as e:
                raise ValueError(f"An error occurred while parsing JSON from the response: {e}")

        else:
            raise ValueError("No valid JSON object could be found in the response.")
    else:
        raise ValueError("The delimiter indicating the start of JSON content was not found.")

def generate_response(invoice_text):
    prompt_template = f'''[INST] <<SYS>>
    You are a helpful, respectful and honest assistant. Always answer as helpfully as possible, while being safe.  Your answers should not include any harmful, unethical, racist, sexist, toxic, dangerous, or illegal content. Please ensure that your responses are socially unbiased and positive in nature. If a question does not make any sense, or is not factually coherent, explain why instead of answering something not correct. If you don't know the answer to a question, please don't share false information.

    Based on the provided invoice text, please extract the necessary information and structure it into the following JSON format:

    {{
        "invoice_number": "",
        "invoice_date": "",
        "due_date": "",
        "total_amount": "",
        "vat_amount": "",
        "client": {{
            "name": "",
            "vat_number": "",
            "address": {{
                "street": "",
                "city": "",
                "postcode": "",
                "country": ""
            }},
            "phone": "",
            "email": ""
        }},
        "supplier": {{
            "name": "",
            "vat_number": "",
            "address": {{
                "street": "",
                "city": "",
                "postcode": "",
                "country": ""
            }},
            "phone": "",
            "email": ""
        }},
        "items": [
            {{
                "description": "",
                "quantity": "",
                "unit_price": "",
                "total": "",
                "vat_rate": ""
            }}
        ],
        "payment_details": {{
            "bank_name": "",
            "iban": "",
            "swift_code": ""
        }}
    }}

    Invoice Text:
    {invoice_text}
    <</SYS>>[/INST]
    '''

    input_ids = tokenizer(prompt_template, return_tensors='pt').input_ids.cuda()
    output = model.generate(inputs=input_ids, temperature=0.7, do_sample=True, top_p=0.95, top_k=40, max_new_tokens=2048)
    return tokenizer.decode(output[0])

@app.route('/format-invoice-info', methods=['POST'])
def format_invoice_info():
    content = request.json
    invoice_text = content['text']
    response = generate_response(invoice_text)
    
    clean_json_response = clean_up_response(response)
    
    return jsonify(json.loads(clean_json_response))

if __name__ == '__main__':
    app.run(debug=True, host='127.0.0.1', port=5000)