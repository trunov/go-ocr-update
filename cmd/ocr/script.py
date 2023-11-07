
from transformers import AutoModelForCausalLM, AutoTokenizer, pipeline
import sys


model_name_or_path = "TheBloke/Llama-2-13B-chat-GPTQ"
# To use a different branch, change revision
# For example: revision="main"
model = AutoModelForCausalLM.from_pretrained(model_name_or_path,
                                             device_map="auto",
                                             trust_remote_code=False,
                                             revision="main")

tokenizer = AutoTokenizer.from_pretrained(model_name_or_path, use_fast=True)

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


# Main loop
while True:
    prompt = sys.stdin.readline().strip()
    response = generate_response(prompt)
    print(response)
    sys.stdout.flush()