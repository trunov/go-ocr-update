from flask import Flask, request, jsonify
from transformers import AutoModelForCausalLM, AutoTokenizer
import json
import logging

logging.basicConfig(filename='app.log', level=logging.DEBUG, format='%(asctime)s %(levelname)s %(name)s %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)

model_name_or_path = "TheBloke/Llama-2-13B-chat-GPTQ"

try:
    logger.debug("Attempting to load the model from path: %s", model_name_or_path)
    model = AutoModelForCausalLM.from_pretrained(model_name_or_path,
                                                 device_map="auto",
                                                 trust_remote_code=False,
                                                 revision="main")
    tokenizer = AutoTokenizer.from_pretrained(model_name_or_path, use_fast=True)
    logger.debug("Model and tokenizer loaded successfully.")
except Exception as e:
    logger.error("Failed to load model or tokenizer: %s", e)
    raise

def clean_up_response(response):
    json_start = response.find("<</SYS>>[/INST]") + len("<</SYS>>[/INST]")
    
    if json_start > len("<</SYS>>[/INST]") - 1:
        json_str = response[json_start:]
        
        start_of_json = json_str.find("{")
        end_of_json = json_str.rfind("}")

        if start_of_json != -1 and end_of_json != -1:
            json_str = json_str[start_of_json:end_of_json+1]

            # Additional handling to remove any extraneous text after the JSON object
            end_of_json_marker = json_str.rfind("}")
            if end_of_json_marker != -1:
                json_str = json_str[:end_of_json_marker + 1]

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
    output = model.generate(inputs=input_ids, temperature=0.7, do_sample=True, top_p=0.95, top_k=40, max_new_tokens=4096)
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